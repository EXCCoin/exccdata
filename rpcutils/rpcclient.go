// Copyright (c) 2018 The ExchangeCoin team
// Copyright (c) 2017, Jonathan Chappelow
// See LICENSE for details.

package rpcutils

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/EXCCoin/exccd/chaincfg"
	"github.com/EXCCoin/exccd/chaincfg/chainhash"
	"github.com/EXCCoin/exccd/exccjson"
	"github.com/EXCCoin/exccd/exccutil"
	"github.com/EXCCoin/exccd/rpcclient"
	"github.com/EXCCoin/exccd/wire"
	apitypes "github.com/EXCCoin/exccdata/api/types"
	"github.com/EXCCoin/exccdata/semver"
	"github.com/EXCCoin/exccdata/txhelpers"
)

// Any of the following exccd RPC API versions are deemed compatible with
// exccdata.
var compatibleChainServerAPIs = []semver.Semver{
	semver.NewSemver(3, 0, 0),
	semver.NewSemver(4, 0, 0), // bumped for removal of createrawssgen, not used by exccdata
}

// ConnectNodeRPC attempts to create a new websocket connection to a exccd node,
// with the given credentials and optional notification handlers.
func ConnectNodeRPC(host, user, pass, cert string, disableTLS bool,
	ntfnHandlers ...*rpcclient.NotificationHandlers) (*rpcclient.Client, semver.Semver, error) {
	var exccdCerts []byte
	var err error
	var nodeVer semver.Semver
	if !disableTLS {
		exccdCerts, err = ioutil.ReadFile(cert)
		if err != nil {
			log.Errorf("Failed to read exccd cert file at %s: %s\n",
				cert, err.Error())
			return nil, nodeVer, err
		}
		log.Debugf("Attempting to connect to exccd RPC %s as user %s "+
			"using certificate located in %s",
			host, user, cert)
	} else {
		log.Debugf("Attempting to connect to exccd RPC %s as user %s (no TLS)",
			host, user)
	}

	connCfgDaemon := &rpcclient.ConnConfig{
		Host:         host,
		Endpoint:     "ws", // websocket
		User:         user,
		Pass:         pass,
		Certificates: exccdCerts,
		DisableTLS:   disableTLS,
	}

	var ntfnHdlrs *rpcclient.NotificationHandlers
	if len(ntfnHandlers) > 0 {
		if len(ntfnHandlers) > 1 {
			return nil, nodeVer, fmt.Errorf("invalid notification handler argument")
		}
		ntfnHdlrs = ntfnHandlers[0]
	}
	exccdClient, err := rpcclient.New(connCfgDaemon, ntfnHdlrs)
	if err != nil {
		return nil, nodeVer, fmt.Errorf("Failed to start exccd RPC client: %s", err.Error())
	}

	// Ensure the RPC server has a compatible API version.
	ver, err := exccdClient.Version()
	if err != nil {
		log.Error("Unable to get RPC version: ", err)
		return nil, nodeVer, fmt.Errorf("unable to get node RPC version")
	}

	exccdVer := ver["exccdjsonrpcapi"]
	nodeVer = semver.NewSemver(exccdVer.Major, exccdVer.Minor, exccdVer.Patch)

	// Check if the exccd RPC API version is compatible with exccdata.
	isApiCompat := semver.AnyCompatible(compatibleChainServerAPIs, nodeVer)
	if !isApiCompat {
		return nil, nodeVer, fmt.Errorf("Node JSON-RPC server does not have "+
			"a compatible API version. Advertises %v but requires one of: %v",
			nodeVer, compatibleChainServerAPIs)
	}

	return exccdClient, nodeVer, nil
}

// BuildBlockHeaderVerbose creates a *exccjson.GetBlockHeaderVerboseResult from
// an input *wire.BlockHeader and current best block height, which is used to
// compute confirmations.  The next block hash may optionally be provided.
func BuildBlockHeaderVerbose(header *wire.BlockHeader, params *chaincfg.Params,
	currentHeight int64, nextHash ...string) *exccjson.GetBlockHeaderVerboseResult {
	if header == nil {
		return nil
	}

	diffRatio := txhelpers.GetDifficultyRatio(header.Bits, params)

	var next string
	if len(nextHash) > 0 {
		next = nextHash[0]
	}

	blockHeaderResult := exccjson.GetBlockHeaderVerboseResult{
		Hash:          header.BlockHash().String(),
		Confirmations: currentHeight - int64(header.Height),
		Version:       header.Version,
		PreviousHash:  header.PrevBlock.String(),
		MerkleRoot:    header.MerkleRoot.String(),
		StakeRoot:     header.StakeRoot.String(),
		VoteBits:      header.VoteBits,
		FinalState:    hex.EncodeToString(header.FinalState[:]),
		Voters:        header.Voters,
		FreshStake:    header.FreshStake,
		Revocations:   header.Revocations,
		PoolSize:      header.PoolSize,
		Bits:          strconv.FormatInt(int64(header.Bits), 16),
		SBits:         exccutil.Amount(header.SBits).ToCoin(),
		Height:        header.Height,
		Size:          header.Size,
		Time:          header.Timestamp.Unix(),
		Nonce:         header.Nonce,
		Difficulty:    diffRatio,
		NextHash:      next,
	}

	return &blockHeaderResult
}

// GetBlockHeaderVerbose creates a *exccjson.GetBlockHeaderVerboseResult for the
// block index specified by idx via an RPC connection to a chain server.
func GetBlockHeaderVerbose(client *rpcclient.Client, params *chaincfg.Params,
	idx int64) *exccjson.GetBlockHeaderVerboseResult {
	blockhash, err := client.GetBlockHash(idx)
	if err != nil {
		log.Errorf("GetBlockHash(%d) failed: %v", idx, err)
		return nil
	}

	blockHeaderVerbose, err := client.GetBlockHeaderVerbose(blockhash)
	if err != nil {
		log.Errorf("GetBlockHeaderVerbose(%v) failed: %v", blockhash, err)
		return nil
	}

	return blockHeaderVerbose
}

// GetBlockVerbose creates a *exccjson.GetBlockVerboseResult for the block index
// specified by idx via an RPC connection to a chain server.
func GetBlockVerbose(client *rpcclient.Client, params *chaincfg.Params,
	idx int64, verboseTx bool) *exccjson.GetBlockVerboseResult {
	blockhash, err := client.GetBlockHash(idx)
	if err != nil {
		log.Errorf("GetBlockHash(%d) failed: %v", idx, err)
		return nil
	}

	blockVerbose, err := client.GetBlockVerbose(blockhash, verboseTx)
	if err != nil {
		log.Errorf("GetBlockVerbose(%v) failed: %v", blockhash, err)
		return nil
	}

	return blockVerbose
}

// GetBlockVerboseByHash creates a *exccjson.GetBlockVerboseResult for the
// specified block hash via an RPC connection to a chain server.
func GetBlockVerboseByHash(client *rpcclient.Client, params *chaincfg.Params,
	hash string, verboseTx bool) *exccjson.GetBlockVerboseResult {
	blockhash, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		log.Errorf("Invalid block hash %s", hash)
		return nil
	}

	blockVerbose, err := client.GetBlockVerbose(blockhash, verboseTx)
	if err != nil {
		log.Errorf("GetBlockVerbose(%v) failed: %v", blockhash, err)
		return nil
	}

	return blockVerbose
}

// GetStakeDiffEstimates combines the results of EstimateStakeDiff and
// GetStakeDifficulty into a *apitypes.StakeDiff.
func GetStakeDiffEstimates(client *rpcclient.Client) *apitypes.StakeDiff {
	stakeDiff, err := client.GetStakeDifficulty()
	if err != nil {
		return nil
	}
	estStakeDiff, err := client.EstimateStakeDiff(nil)
	if err != nil {
		return nil
	}
	stakeDiffEstimates := apitypes.StakeDiff{
		GetStakeDifficultyResult: exccjson.GetStakeDifficultyResult{
			CurrentStakeDifficulty: stakeDiff.CurrentStakeDifficulty,
			NextStakeDifficulty:    stakeDiff.NextStakeDifficulty,
		},
		Estimates: *estStakeDiff,
	}
	return &stakeDiffEstimates
}

// GetBlock gets a block at the given height from a chain server.
func GetBlock(ind int64, client *rpcclient.Client) (*exccutil.Block, *chainhash.Hash, error) {
	blockhash, err := client.GetBlockHash(ind)
	if err != nil {
		return nil, nil, fmt.Errorf("GetBlockHash(%d) failed: %v", ind, err)
	}

	msgBlock, err := client.GetBlock(blockhash)
	if err != nil {
		return nil, blockhash,
			fmt.Errorf("GetBlock failed (%s): %v", blockhash, err)
	}
	block := exccutil.NewBlock(msgBlock)

	return block, blockhash, nil
}

// GetBlockByHash gets the block with the given hash from a chain server.
func GetBlockByHash(blockhash *chainhash.Hash, client *rpcclient.Client) (*exccutil.Block, error) {
	msgBlock, err := client.GetBlock(blockhash)
	if err != nil {
		return nil, fmt.Errorf("GetBlock failed (%s): %v", blockhash, err)
	}
	block := exccutil.NewBlock(msgBlock)

	return block, nil
}

// GetTransactionVerboseByID get a transaction by transaction id
func GetTransactionVerboseByID(client *rpcclient.Client, txid string) (*exccjson.TxRawResult, error) {
	txhash, err := chainhash.NewHashFromStr(txid)
	if err != nil {
		log.Errorf("Invalid transaction hash %s", txid)
		return nil, err
	}

	txraw, err := client.GetRawTransactionVerbose(txhash)
	if err != nil {
		log.Errorf("GetRawTransactionVerbose failed for: %v", txhash)
		return nil, err
	}
	return txraw, nil
}

// SearchRawTransaction fetch transactions the belong to an
// address
func SearchRawTransaction(client *rpcclient.Client, count int, address string) ([]*exccjson.SearchRawTransactionsResult, error) {
	addr, err := exccutil.DecodeAddress(address)
	if err != nil {
		log.Infof("Invalid address %s: %v", address, err)
		return nil, err
	}
	//change the 1000 000 number demo for now
	txs, err := client.SearchRawTransactionsVerbose(addr, 0, count,
		true, true, nil)
	if err != nil {
		log.Warnf("SearchRawTransaction failed for address %s: %v", addr, err)
	}
	return txs, nil
}
