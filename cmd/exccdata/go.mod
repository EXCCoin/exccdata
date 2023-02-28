module github.com/EXCCoin/exccdata/cmd/exccdata

go 1.18

replace (
	github.com/EXCCoin/exccdata/db/dcrpg/v8 => ../../db/dcrpg/
	github.com/EXCCoin/exccdata/exchanges/v3 => ../../exchanges/
	github.com/EXCCoin/exccdata/gov/v6 => ../../gov/
	github.com/EXCCoin/exccdata/v8 => ../../
)

