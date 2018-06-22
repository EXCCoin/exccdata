# exccdata

[![Build Status](https://img.shields.io/travis/EXCCoin/exccdata.svg)](https://travis-ci.org/EXCCoin/exccdata)
[![GitHub release](https://img.shields.io/github/release/EXCCoin/exccdata.svg)](https://github.com/EXCCoin/exccdata/releases)
[![Latest tag](https://img.shields.io/github/tag/EXCCoin/exccdata.svg)](https://github.com/EXCCoin/exccdata/tags)
[![Go Report Card](https://goreportcard.com/badge/github.com/EXCCoin/exccdata)](https://goreportcard.com/report/github.com/EXCCoin/exccdata)
[![ISC License](https://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

The exccdata repository is a collection of golang packages and apps for [EXCCoin](https://excc.co/) data collection, storage, and presentation.

## Repository overview

```none
../exccdata             The exccdata daemon.
├── blockdata           Package blockdata.
├── cmd
│   ├── rebuilddb       rebuilddb utility, for SQLite backend.
│   ├── rebuilddb2      rebuilddb2 utility, for PostgreSQL backend.
│   └── scanblocks      scanblocks utility.
├── db
│   ├── dbtypes         Package dbtypes with common data types.
│   ├── dcrpg           Package dcrpg providing PostgreSQL backend.
│   └── dcrsqlite       Package dcrsqlite providing SQLite backend.
├── dev                 Shell scripts for maintenance and deployment.
├── public              Public resources for block explorer (css, js, etc.).
├── explorer            Package explorer, powering the block explorer.
├── mempool             Package mempool.
├── rpcutils            Package rpcutils.
├── semver              Package semver.
├── stakedb             Package stakedb, for tracking tickets.
├── txhelpers           Package txhelpers.
└── views               HTML templates for block explorer.
```

## Requirements

* [Go](http://golang.org) 1.9.x or 1.10.x.
* Running `exccd` (>=1.1.2) synchronized to the current best block on the network.

## Installation

### Build from Source

The following instructions assume a Unix-like shell (e.g. bash).

* [Install Go](http://golang.org/doc/install)

* Verify Go installation:

      go env GOROOT GOPATH

* Ensure `$GOPATH/bin` is on your `$PATH`.
* Install `dep`, the dependency management tool.

      go get -u -v github.com/golang/dep/cmd/dep

* Clone the exccdata repository. It **must** be cloned into the following directory.

      git clone https://github.com/EXCCoin/exccdata $GOPATH/src/github.com/EXCCoin/exccdata

* Fetch dependencies, and build the `exccdata` executable.

      cd $GOPATH/src/github.com/EXCCoin/exccdata
      dep ensure
      # build exccdata executable in workspace:
      go build

The sqlite driver uses cgo, which requires a C compiler (e.g. gcc) to compile the C sources. On
Windows this is easily handled with MSYS2 ([download](http://www.msys2.org/) and
install MinGW-w64 gcc packages).

Tip: If you receive other build errors, it may be due to "vendor" directories
left by dep builds of dependencies such as exccwallet. You may safely delete
vendor folders and run `dep ensure` again.

### Runtime resources

The config file, logs, and data files are stored in the application data folder, which may be specified via the `-A/--appdata` setting. However, the location of the config file may be set with `-C/--configfile`.

The "public" and "views" folders *must* be in the same
folder as the `exccdata` executable.

## Updating

First, update the repository (assuming you have `master` checked out):

    cd $GOPATH/src/github.com/EXCCoin/exccdata
    git pull origin master
    dep ensure
    go build

Look carefully for errors with `git pull`, and reset locally modified files if
necessary.

## Getting Started

### Configure PostgreSQL (IMPORTANT)

If you intend to run exccdata in "full" mode (i.e. with the `--pg` switch), which
uses a PostgreSQL database backend, it is crucial that you configure your
PostgreSQL server for your hardware and the exccdata workload.

Read [postgresql-tuning.conf](./db/dcrpg/postgresql-tuning.conf) carefully for
details on how to make the necessary changes to your system.

### Create configuration file

Begin with the sample configuration file:

```bash
cp sample-exccdata.conf exccdata.conf
```

Then edit exccdata.conf with your exccd RPC settings. After you are finished, move
exccdata.conf to the `appdata` folder (default is `~/.exccdata` on Linux,
`%localappdata%\exccdata` on Windows). See the output of `exccdata --help` for a list
of all options and their default values.

### Indexing the Blockchain

If exccdata has not previously been run with the PostgreSQL database backend, it
is necessary to perform a bulk import of blockchain data and generate table
indexes. *This will be done automatically by `exccdata`* on a fresh startup.

Alternatively, the PostgreSQL tables may also be generated with the `rebuilddb2`
command line tool:

* Create the exccdata user and database in PostgreSQL (tables will be created automatically).
* Set your PostgreSQL credentials and host in both `./cmd/rebuilddb2/rebuilddb2.conf`,
  and `exccdata.conf` in the location specified by the `appdata` flag.
* Run `./rebuilddb2` to bulk import data and index the tables.
* In case of irrecoverable errors, such as detected schema changes without an
  upgrade path, the tables and their indexes may be dropped with `rebuilddb2 -D`.

Note that exccdata requires that [exccd](https://github.com/EXCCoin/exccd/blob/master/README.md) is running with optional indexes enabled.  By default these indexes are not turned on when exccd is installed.

In exccd.conf set:
```
txindex=1
addrindex=1
```

### Starting exccdata

Launch the exccdata daemon and allow the databases to process new blocks. Both
SQLite and PostgreSQL synchronization require about an hour the first time
exccdata is run, but they are done concurrently. On subsequent launches, only
blocks new to exccdata are processed.

```bash
./exccdata    # don't forget to configure exccdata.conf in the appdata folder!
```

Unlike exccdata.conf, which must be placed in the `appdata` folder or explicitly
set with `-C`, the "public" and "views" folders *must* be in the same folder as
the `exccdata` executable.

## exccdata daemon

The root of the repository is the `main` package for the exccdata app, which has
several components including:

1. Block explorer (web interface).
1. Blockchain monitoring and data collection.
1. Mempool monitoring and reporting.
1. Data storage in durable database (sqlite presently).
1. RESTful JSON API over HTTP(S).

### Block Explorer

After exccdata syncs with the blockchain server via RPC, by default it will begin
listening for HTTP connections on `http://127.0.0.1:7777/`. This means it starts
a web server listening on IPv4 localhost, port 7777. Both the interface and port
are configurable. The block explorer and the JSON API are both provided by the
server on this port. See [JSON REST API](#json-rest-api) for details.

Note that while exccdata can be started with HTTPS support, it is recommended to
employ a reverse proxy such as nginx. See sample-nginx.conf for an example nginx
configuration.

A new auxillary database backend using PostgreSQL was introduced in v0.9.0 that
provides expanded functionality. However, initial population of the database
takes additional time and tens of gigabytes of disk storage space. Thus, exccdata
runs by default in a reduced functionality mode that does not require
PostgreSQL. To enable the PostgreSQL backend (and the expanded functionality),
exccdata may be started with the `--pg` switch.

### JSON REST API

The API serves JSON data over HTTP(S). **All API endpoints are currently
prefixed with `/api`** (e.g. `http://localhost:7777/api/stake`).

#### Endpoint List

| Best block | Path | Type |
| --- | --- | --- |
| Summary | `/block/best` | `types.BlockDataBasic` |
| Stake info |  `/block/best/pos` | `types.StakeInfoExtended` |
| Header |  `/block/best/header` | `exccjson.GetBlockHeaderVerboseResult` |
| Hash |  `/block/best/hash` | `string` |
| Height | `/block/best/height` | `int` |
| Size | `/block/best/size` | `int32` |
| Transactions | `/block/best/tx` | `types.BlockTransactions` |
| Transactions Count | `/block/best/tx/count` | `types.BlockTransactionCounts` |
| Verbose block result | `/block/best/verbose` | `exccjson.GetBlockVerboseResult` |

| Block X (block index) | Path | Type |
| --- | --- | --- |
| Summary | `/block/X` | `types.BlockDataBasic` |
| Stake info |  `/block/X/pos` | `types.StakeInfoExtended` |
| Header |  `/block/X/header` | `exccjson.GetBlockHeaderVerboseResult` |
| Hash |  `/block/X/hash` | `string` |
| Size | `/block/X/size` | `int32` |
| Transactions | `/block/X/tx` | `types.BlockTransactions` |
| Transactions Count | `/block/X/tx/count` | `types.BlockTransactionCounts` |
| Verbose block result | `/block/X/verbose` | `exccjson.GetBlockVerboseResult` |

| Block H (block hash) | Path | Type |
| --- | --- | --- |
| Summary | `/block/hash/H` | `types.BlockDataBasic` |
| Stake info |  `/block/hash/H/pos` | `types.StakeInfoExtended` |
| Header |  `/block/hash/H/header` | `exccjson.GetBlockHeaderVerboseResult` |
| Height |  `/block/hash/H/height` | `int` |
| Size | `/block/hash/H/size` | `int32` |
| Transactions | `/block/hash/H/tx` | `types.BlockTransactions` |
| Transactions Count | `/block/hash/H/tx/count` | `types.BlockTransactionCounts` |
| Verbose block result | `/block/hash/H/verbose` | `exccjson.GetBlockVerboseResult` |

| Block range (X < Y) | Path | Type |
| --- | --- | --- |
| Summary array for blocks on `[X,Y]` | `/block/range/X/Y` | `[]types.BlockDataBasic` |
| Summary array with block index step `S` | `/block/range/X/Y/S` | `[]types.BlockDataBasic` |
| Size (bytes) array | `/block/range/X/Y/size` | `[]int32` |
| Size array with step `S` | `/block/range/X/Y/S/size` | `[]int32` |

| Transaction T (transaction id) | Path | Type |
| --- | --- | --- |
| Transaction Details | `/tx/T` | `types.Tx` |
| Inputs | `/tx/T/in` | `[]types.TxIn` |
| Details for input at index `X` | `/tx/T/in/X` | `types.TxIn` |
| Outputs | `/tx/T/out` | `[]types.TxOut` |
| Details for output at index `X` | `/tx/T/out/X` | `types.TxOut` |

| Address A | Path | Type |
| --- | --- | --- |
| Summary of last 10 transactions | `/address/A` | `types.Address` |
| Verbose transaction result for last <br> 10 transactions | `/address/A/raw` | `types.AddressTxRaw` |
| Summary of last `N` transactions | `/address/A/count/N` | `types.Address` |
| Verbose transaction result for last <br> `N` transactions | `/address/A/count/N/raw` | `types.AddressTxRaw` |
| Summary of last `N` transactions, skipping `M` | `/address/A/count/N/skip/M` | `types.Address` |
| Verbose transaction result for last <br> `N` transactions, skipping `M` | `/address/A/count/N/skip/Mraw` | `types.AddressTxRaw` |

| Stake Difficulty (Ticket Price) | Path | Type |
| --- | --- | --- |
| Current sdiff and estimates | `/stake/diff` | `types.StakeDiff` |
| Sdiff for block `X` | `/stake/diff/b/X` | `[]float64` |
| Sdiff for block range `[X,Y] (X <= Y)` | `/stake/diff/r/X/Y` | `[]float64` |
| Current sdiff separately | `/stake/diff/current` | `exccjson.GetStakeDifficultyResult` |
| Estimates separately | `/stake/diff/estimates` | `exccjson.EstimateStakeDiffResult` |

| Ticket Pool | Path | Type |
| --- | --- | --- |
| Current pool info (size, total value, and average price) | `/stake/pool` | `types.TicketPoolInfo` |
| Current ticket pool, in a JSON object with a `"tickets"` key holding an array of ticket hashes | `/stake/pool/full` | `[]string` |
| Pool info for block `X` | `/stake/pool/b/X` | `types.TicketPoolInfo` |
| Full ticket pool at block height _or_ hash `H` | `/stake/pool/b/H/full` | `[]string` |
| Pool info for block range `[X,Y] (X <= Y)` | `/stake/pool/r/X/Y?arrays=[true\|false]`<sup>*</sup> | `[]apitypes.TicketPoolInfo` |

The full ticket pool endpoints accept the URL query `?sort=[true\|false]` for
requesting the tickets array in lexicographical order.  If a sorted list or list
with deterministic order is _not_ required, using `sort=false` will reduce
server load and latency. However, be aware that the ticket order will be random,
and will change each time the tickets are requested.

<sup>*</sup>For the pool info block range endpoint that accepts the `arrays` url query,
a value of `true` will put all pool values and pool sizes into separate arrays,
rather than having a single array of pool info JSON objects.  This may make
parsing more efficient for the client.

| Vote and Agenda Info | Path | Type |
| --- | --- | --- |
| The current agenda and its status | `/stake/vote/info` | `exccjson.GetVoteInfoResult` |

| Mempool | Path | Type |
| --- | --- | --- |
| Ticket fee rate summary | `/mempool/sstx` | `apitypes.MempoolTicketFeeInfo` |
| Ticket fee rate list (all) | `/mempool/sstx/fees` | `apitypes.MempoolTicketFees` |
| Ticket fee rate list (N highest) | `/mempool/sstx/fees/N` | `apitypes.MempoolTicketFees` |
| Detailed ticket list (fee, hash, size, age, etc.) | `/mempool/sstx/details` | `apitypes.MempoolTicketDetails` |
| Detailed ticket list (N highest fee rates) | `/mempool/sstx/details/N`| `apitypes.MempoolTicketDetails` |

| Other | Path | Type |
| --- | --- | --- |
| Status | `/status` | `types.Status` |
| Coin Supply | `/supply` | `types.CoinSupply` |
| Endpoint list (always indented) | `/list` | `[]string` |
| Directory | `/directory` | `string` |

All JSON endpoints accept the URL query `indent=[true|false]`.  For example,
`/stake/diff?indent=true`. By default, indentation is off. The characters to use
for indentation may be specified with the `indentjson` string configuration
option.

## Important Note About Mempool

Although there is mempool data collection and serving, it is **very important**
to keep in mind that the mempool in your node (exccd) is not likely to be exactly
the same as other nodes' mempool.  Also, your mempool is cleared out when you
shutdown exccd.  So, if you have recently (e.g. after the start of the current
ticket price window) started exccd, your mempool _will_ be missing transactions
that other nodes have.

## Command Line Utilities

### rebuilddb

`rebuilddb` is a CLI app that performs a full blockchain scan that fills past
block data into a SQLite database. This functionality is included in the startup
of the exccdata daemon, but may be called alone with rebuilddb.

### rebuilddb2

`rebuilddb2` is a CLI app used for maintenance of exccdata's `dcrpg` database
(a.k.a. DB v2) that uses PostgreSQL to store a nearly complete record of the
EXCCoin blockchain data. See the [README.md](./cmd/rebuilddb2/README.md) for
`rebuilddb2` for important usage information.

### scanblocks

scanblocks is a CLI app to scan the blockchain and save data into a JSON file.
More details are in [its own README](./cmd/scanblocks/README.md). The repository
also includes a shell script, jsonarray2csv.sh, to convert the result into a
comma-separated value (CSV) file.

## Helper packages

`package dbtypes` defines the data types used by the DB backends to model the
block, transaction, and related blockchain data structures. Functions for
converting from standard EXCCoin data types (e.g. `wire.MsgBlock`) are also
provided.

`package rpcutils` includes helper functions for interacting with a
`rpcclient.Client`.

`package stakedb` defines the `StakeDatabase` and `ChainMonitor` types for
efficiently tracking live tickets, with the primary purpose of computing ticket
pool value quickly.  It uses the `database.DB` type from
`github.com/EXCCoin/exccdata/database` with an ffldb storage backend from
`github.com/EXCCoin/exccdata/database/ffldb`.  It also makes use of the `stake.Node`
type from `github.com/EXCCoin/exccdata/blockchain/stake`.  The `ChainMonitor` type
handles connecting new blocks and chain reorganization in response to notifications
from exccd.

`package txhelpers` includes helper functions for working with the common types
`exccutil.Tx`, `exccutil.Block`, `chainhash.Hash`, and others.

## Internal-use packages

Packages `blockdata` and `dcrsqlite` are currently designed only for internal
use internal use by other exccdata packages, but they may be of general value in
the future.

`blockdata` defines:

* The `chainMonitor` type and its `BlockConnectedHandler()` method that handles
  block-connected notifications and triggers data collection and storage.
* The `BlockData` type and methods for converting to API types.
* The `blockDataCollector` type and its `Collect()` and `CollectHash()` methods
  that are called by the chain monitor when a new block is detected.
* The `BlockDataSaver` interface required by `chainMonitor` for storage of
  collected data.

`dcrpg` defines:

* The `ChainDB` type, which is the primary exported type from `dcrpg`, providing
  an interface for a PostgreSQL database.
* A large set of lower-level functions to perform a range of queries given a
  `*sql.DB` instance and various parameters.
* The internal package contains the raw SQL statements.

`dcrsqlite` defines:

* A `sql.DB` wrapper type (`DB`) with the necessary SQLite queries for
  storage and retrieval of block and stake data.
* The `wiredDB` type, intended to satisfy the `DataSourceLite` interface used by
  the exccdata app's API. The block header is not stored in the DB, so a RPC
  client is used by `wiredDB` to get it on demand. `wiredDB` also includes
  methods to resync the database file.

`package mempool` defines a `mempoolMonitor` type that can monitor a node's
mempool using the `OnTxAccepted` notification handler to send newly received
transaction hashes via a designated channel. Ticket purchases (SSTx) are
triggers for mempool data collection, which is handled by the
`mempoolDataCollector` class, and data storage, which is handled by any number
of objects implementing the `MempoolDataSaver` interface.

## Plans

See the GitHub issue tracker and the [project milestones](https://github.com/EXCCoin/exccdata/milestones).

## Contributing

Yes, please! See the CONTRIBUTING.md file for details, but here's the gist of it:

1. Fork the repo.
1. Create a branch for your work (`git branch -b cool-stuff`).
1. Code something great.
1. Commit and push to your repo.
1. Create a [pull request](https://github.com/EXCCoin/exccdata/compare).

Before committing any changes to the Gopkg.lock file, you must update `dep` to
the latest version via:

    go get -u github.com/golang/dep/cmd/dep

**To update `dep` from the network, it is important to use the `-u` flag as
shown above.**

Note that all EXCCoin community and team members are expected to adhere to
the code of conduct, described in the CODE_OF_CONDUCT file.

## License

This project is licensed under the ISC License. See the [LICENSE](LICENSE) file for details.
