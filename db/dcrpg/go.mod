module github.com/EXCCoin/exccdata/db/dcrpg/v8

go 1.18

replace github.com/EXCCoin/exccdata/v8 => ../../

require (
	github.com/EXCCoin/exccd/blockchain/stake/v4 v4.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/chaincfg/chainhash v0.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/chaincfg/v3 v3.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/dcrutil/v4 v4.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/rpc/jsonrpc/types/v3 v3.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/rpcclient/v7 v7.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/txscript/v4 v4.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccd/wire v0.0.0-20231114094815-cf10ea2807e7
	github.com/EXCCoin/exccdata/v8 v8.0.0-20230419111953-ae3472cbd807
	github.com/davecgh/go-spew v1.1.1
	github.com/decred/slog v1.2.0
	github.com/dustin/go-humanize v1.0.1
	github.com/jessevdk/go-flags v1.5.0
	github.com/jrick/logrotate v1.0.0
	github.com/lib/pq v1.10.9
)

require (
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/EXCCoin/base58 v0.0.0-20180515090142-e1a805ee5d9f // indirect
	github.com/EXCCoin/exccd v0.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/blockchain/standalone/v2 v2.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/crypto/blake256 v0.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/crypto/ripemd160 v0.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/database/v3 v3.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/dcrec v0.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/dcrec/edwards/v2 v2.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/dcrec/secp256k1/v4 v4.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/dcrjson/v4 v4.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/EXCCoin/exccd/gcs/v3 v3.0.0-20231114094815-cf10ea2807e7 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/decred/go-socks v1.1.0 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
