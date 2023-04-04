module github.com/EXCCoin/exccdata/gov/v6

go 1.18

replace github.com/EXCCoin/exccdata/v8 => ../

require (
	github.com/EXCCoin/exccd/chaincfg/v3 v3.0.0-20230329171406-0c77daae7811
	github.com/EXCCoin/exccd/dcrjson/v4 v4.0.0-20230329171406-0c77daae7811
	github.com/EXCCoin/exccd/rpc/jsonrpc/types/v3 v3.0.0-20230329171406-0c77daae7811
	github.com/EXCCoin/exccdata/v8 v8.0.0-20230228130636-8644fde8d585
	github.com/asdine/storm/v3 v3.2.1
	github.com/decred/slog v1.2.0
)

require (
	github.com/EXCCoin/base58 v0.0.0-20180515090142-e1a805ee5d9f // indirect
	github.com/EXCCoin/exccd v0.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/blockchain/stake/v4 v4.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/blockchain/standalone/v2 v2.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/chaincfg/chainhash v0.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/crypto/blake256 v0.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/crypto/ripemd160 v0.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/database/v3 v3.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/dcrec v0.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/dcrec/edwards/v2 v2.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/dcrec/secp256k1/v4 v4.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/dcrutil/v4 v4.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/txscript/v4 v4.0.0-20230329171406-0c77daae7811 // indirect
	github.com/EXCCoin/exccd/wire v0.0.0-20230329171406-0c77daae7811 // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	golang.org/x/crypto v0.7.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
)
