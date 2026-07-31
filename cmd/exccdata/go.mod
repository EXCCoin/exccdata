module github.com/EXCCoin/exccdata/cmd/exccdata

go 1.25.0

replace (
	github.com/EXCCoin/exccdata/db/dcrpg/v8 => ../../db/dcrpg/
	github.com/EXCCoin/exccdata/exchanges/v3 => ../../exchanges/
	github.com/EXCCoin/exccdata/gov/v6 => ../../gov/
	github.com/EXCCoin/exccdata/v8 => ../../
)

require (
	github.com/EXCCoin/exccd/blockchain/stake/v4 v4.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/chaincfg/chainhash v0.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/chaincfg/v3 v3.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/dcrutil/v4 v4.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/rpc/jsonrpc/types/v3 v3.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/rpcclient/v7 v7.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/txscript/v4 v4.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccd/wire v0.0.0-20260731082039-ab9b75d61981
	github.com/EXCCoin/exccdata/db/dcrpg/v8 v8.0.0-20230419111953-ae3472cbd807
	github.com/EXCCoin/exccdata/exchanges/v3 v3.1.0
	github.com/EXCCoin/exccdata/gov/v6 v6.0.0-20230419111953-ae3472cbd807
	github.com/EXCCoin/exccdata/v8 v8.0.0-20230419111953-ae3472cbd807
	github.com/caarlos0/env/v6 v6.10.1
	github.com/decred/slog v1.2.0
	github.com/didip/tollbooth/v6 v6.1.2
	github.com/dustin/go-humanize v1.0.1
	github.com/go-chi/chi/v5 v5.3.0
	github.com/go-chi/docgen v1.2.0
	github.com/google/gops v0.3.28
	github.com/googollee/go-socket.io v1.7.0
	github.com/jessevdk/go-flags v1.5.0
	github.com/jrick/logrotate v1.0.0
	github.com/rs/cors v1.11.0
	golang.org/x/net v0.56.0
	golang.org/x/text v0.39.0
)

require (
	decred.org/cspp/v2 v2.4.0 // indirect
	decred.org/dcrdex v1.0.4 // indirect
	decred.org/dcrwallet/v4 v4.3.1 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20190825152654-46b345b51c96 // indirect
	github.com/EXCCoin/base58 v0.0.0-20180515090142-e1a805ee5d9f // indirect
	github.com/EXCCoin/exccd v0.0.0-20260731082039-ab9b75d61981 // indirect
	github.com/EXCCoin/exccd/blockchain/standalone/v2 v2.0.0-20260731082039-ab9b75d61981 // indirect
	github.com/EXCCoin/exccd/crypto/blake256 v0.0.0-20260730143238-5ec2339c687a // indirect
	github.com/EXCCoin/exccd/crypto/ripemd160 v0.0.0-20260730143238-5ec2339c687a // indirect
	github.com/EXCCoin/exccd/database/v3 v3.0.0-20260731082039-ab9b75d61981 // indirect
	github.com/EXCCoin/exccd/dcrec v0.0.0-20260730143238-5ec2339c687a // indirect
	github.com/EXCCoin/exccd/dcrec/edwards/v2 v2.0.0-20260730143238-5ec2339c687a // indirect
	github.com/EXCCoin/exccd/dcrec/secp256k1/v4 v4.0.0-20260730143238-5ec2339c687a // indirect
	github.com/EXCCoin/exccd/dcrjson/v4 v4.0.0-20260731082039-ab9b75d61981 // indirect
	github.com/EXCCoin/exccd/gcs/v3 v3.0.0-20260730143238-5ec2339c687a // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProjectZKM/Ziren/crates/go-runtime/zkvm_runtime v0.0.0-20251001021608-1fe7b43fc4d6 // indirect
	github.com/VictoriaMetrics/fastcache v1.13.0 // indirect
	github.com/aead/siphash v1.0.1 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/asdine/storm/v3 v3.2.1 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/btcsuite/btcd v0.24.2-beta.rc1.0.20240625142744-cc26860b4026 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.5 // indirect
	github.com/btcsuite/btcd/btcutil/psbt v1.1.8 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0 // indirect
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f // indirect
	github.com/btcsuite/btcwallet v0.16.10 // indirect
	github.com/btcsuite/btcwallet/wallet/txauthor v1.3.5 // indirect
	github.com/btcsuite/btcwallet/wallet/txrules v1.2.2 // indirect
	github.com/btcsuite/btcwallet/wallet/txsizes v1.2.5 // indirect
	github.com/btcsuite/btcwallet/walletdb v1.4.4 // indirect
	github.com/btcsuite/btcwallet/wtxmgr v1.5.4 // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd // indirect
	github.com/btcsuite/golangcrypto v0.0.0-20150304025918-53f62d9b43e8 // indirect
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792 // indirect
	github.com/carterjones/go-cloudflare-scraper v0.1.2 // indirect
	github.com/carterjones/signalr v0.3.5 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/companyzero/sntrup4591761 v0.0.0-20220309191932-9e0f3af2f07a // indirect
	github.com/consensys/gnark-crypto v0.18.1 // indirect
	github.com/crate-crypto/go-eth-kzg v1.4.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/blake2b v1.0.0 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/dcrlabs/bchwallet v0.0.0-20240114124852-0e95005810be // indirect
	github.com/dcrlabs/ltcwallet v0.0.0-20240823165752-3e026e8da010 // indirect
	github.com/dcrlabs/neutrino-bch v0.0.0-20240114121828-d656bce11095 // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/base58 v1.0.5 // indirect
	github.com/decred/dcrd/addrmgr/v2 v2.0.4 // indirect
	github.com/decred/dcrd/blockchain/stake/v5 v5.0.1 // indirect
	github.com/decred/dcrd/blockchain/standalone/v2 v2.2.1 // indirect
	github.com/decred/dcrd/certgen v1.2.0 // indirect
	github.com/decred/dcrd/chaincfg/chainhash v1.0.4 // indirect
	github.com/decred/dcrd/chaincfg/v3 v3.2.1 // indirect
	github.com/decred/dcrd/connmgr/v3 v3.1.2 // indirect
	github.com/decred/dcrd/container/lru v1.0.0 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.1.0 // indirect
	github.com/decred/dcrd/crypto/rand v1.0.1 // indirect
	github.com/decred/dcrd/crypto/ripemd160 v1.0.2 // indirect
	github.com/decred/dcrd/database/v3 v3.0.2 // indirect
	github.com/decred/dcrd/dcrec v1.0.1 // indirect
	github.com/decred/dcrd/dcrec/edwards/v2 v2.0.3 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/decred/dcrd/dcrjson/v4 v4.1.0 // indirect
	github.com/decred/dcrd/dcrutil/v4 v4.0.2 // indirect
	github.com/decred/dcrd/gcs/v4 v4.1.0 // indirect
	github.com/decred/dcrd/hdkeychain/v3 v3.1.2 // indirect
	github.com/decred/dcrd/lru v1.1.2 // indirect
	github.com/decred/dcrd/mixing v0.5.0 // indirect
	github.com/decred/dcrd/rpc/jsonrpc/types/v4 v4.3.0 // indirect
	github.com/decred/dcrd/rpcclient/v8 v8.0.1 // indirect
	github.com/decred/dcrd/txscript/v4 v4.1.1 // indirect
	github.com/decred/dcrd/wire v1.7.0 // indirect
	github.com/decred/go-socks v1.1.0 // indirect
	github.com/decred/vspd/client/v4 v4.0.1 // indirect
	github.com/decred/vspd/types/v2 v2.1.0 // indirect
	github.com/decred/vspd/types/v3 v3.0.0 // indirect
	github.com/dgraph-io/badger v1.6.2 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/emicklei/dot v1.6.2 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.5 // indirect
	github.com/ethereum/go-bigmodexpfix v0.0.0-20250911101455-f9e208c548ab // indirect
	github.com/ethereum/go-ethereum v1.17.0 // indirect
	github.com/ferranbt/fastssz v0.1.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gcash/bchd v0.19.0 // indirect
	github.com/gcash/bchlog v0.0.0-20180913005452-b4f036f92fa6 // indirect
	github.com/gcash/bchutil v0.0.0-20210113190856-6ea28dff4000 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-pkgz/expirable-cache v1.0.0 // indirect
	github.com/gofrs/flock v0.12.1 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/gomodule/redigo v1.8.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/holiman/billy v0.0.0-20250707135307-f2f9b9aae7db // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/jrick/bitset v1.0.0 // indirect
	github.com/jrick/wsrpc/v2 v2.3.8 // indirect
	github.com/kkdai/bstream v1.0.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/lib/pq v1.12.3 // indirect
	github.com/lightninglabs/gozmq v0.0.0-20191113021534-d20a764486bf // indirect
	github.com/lightninglabs/neutrino v0.16.1-0.20240814152458-81d6cd2d2da5 // indirect
	github.com/lightninglabs/neutrino/cache v1.1.2 // indirect
	github.com/lightningnetwork/lnd/clock v1.1.1 // indirect
	github.com/lightningnetwork/lnd/queue v1.1.1 // indirect
	github.com/lightningnetwork/lnd/ticker v1.1.1 // indirect
	github.com/lightningnetwork/lnd/tlv v1.1.2 // indirect
	github.com/ltcsuite/lnd/clock v1.1.0 // indirect
	github.com/ltcsuite/lnd/queue v1.1.0 // indirect
	github.com/ltcsuite/lnd/ticker v1.1.0 // indirect
	github.com/ltcsuite/lnd/tlv v0.0.0-20240222214433-454d35886119 // indirect
	github.com/ltcsuite/ltcd v0.23.6-0.20240131072528-64dfa402637a // indirect
	github.com/ltcsuite/ltcd/btcec/v2 v2.3.2 // indirect
	github.com/ltcsuite/ltcd/chaincfg/chainhash v1.0.2 // indirect
	github.com/ltcsuite/ltcd/ltcutil v1.1.4-0.20240131072528-64dfa402637a // indirect
	github.com/ltcsuite/ltcd/ltcutil/psbt v1.1.1-0.20240131072528-64dfa402637a // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robertkrimen/otto v0.2.1 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/supranational/blst v0.3.16-0.20250831170142-f48500c1fdbe // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	github.com/zquestz/grab v0.0.0-20190224022517-abcee96e61b1 // indirect
	go.etcd.io/bbolt v1.3.11 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/term v0.44.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.3.0 // indirect
)
