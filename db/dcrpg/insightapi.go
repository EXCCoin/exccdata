// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package dcrpg

import (
	"github.com/EXCCoin/exccd/exccjson"
	"github.com/EXCCoin/exccd/exccutil"
	apitypes "github.com/EXCCoin/exccdata/api/types"
	"github.com/EXCCoin/exccdata/db/dbtypes"
	"github.com/EXCCoin/exccdata/explorer"
	"github.com/EXCCoin/exccdata/rpcutils"
	"github.com/EXCCoin/exccdata/txhelpers"
)

// GetRawTransaction gets a dcrjson.TxRawResult for the specified transaction
// hash.
func (pgb *ChainDBRPC) GetRawTransaction(txid string) (*dcrjson.TxRawResult, error) {
	txraw, err := rpcutils.GetTransactionVerboseByID(pgb.Client, txid)
	if err != nil {
		log.Errorf("GetRawTransactionVerbose failed for: %s", txid)
		return nil, err
	}
	return txraw, nil
}

// GetBlockHeight returns the height of the block with the specified hash.
func (pgb *ChainDB) GetBlockHeight(hash string) (int64, error) {
	height, err := RetrieveBlockHeight(pgb.db, hash)
	if err != nil {
		log.Errorf("Unable to get block height for hash %s: %v", hash, err)
		return -1, err
	}
	return height, nil
}

// GetHeight returns the current best block height.
func (pgb *ChainDB) GetHeight() int {
	height, _, _, _ := RetrieveBestBlockHeight(pgb.db)
	return int(height)
}

// SendRawTransaction attempts to decode the input serialized transaction,
// passed as hex encoded string, and broadcast it, returning the tx hash.
func (db *ChainDBRPC) SendRawTransaction(txhex string) (string, error) {
	msg, err := txhelpers.MsgTxFromHex(txhex)
	if err != nil {
		log.Errorf("SendRawTransaction failed: could not decode hex")
		return "", err
	}
	hash, err := db.Client.SendRawTransaction(msg, true)
	if err != nil {
		log.Errorf("SendRawTransaction failed: %v", err)
		return "", err
	}
	return hash.String(), err
}

// InsightPgGetAddressTransactions performs a db query to pull all txids for the
// specified addresses ordered desc by time.
func (pgb *ChainDB) InsightPgGetAddressTransactions(addr []string,
	recentBlockHeight int64) ([]string, []string) {
	return RetrieveAddressTxnsOrdered(pgb.db, addr, recentBlockHeight)
}

// Update Vin due to DCRD AMOUNTIN - START
func (pgb *ChainDB) RetrieveAddressIDsByOutpoint(txHash string,
	voutIndex uint32) ([]uint64, []string, int64, error) {
	return RetrieveAddressIDsByOutpoint(pgb.db, txHash, voutIndex)
} // Update Vin due to DCRD AMOUNTIN - END

// InsightGetAddressTransactions performs a searchrawtransactions for the
// specfied address, max number of transactions, and offset into the transaction
// list. The search results are in reverse temporal order.
// TODO: Does this really need all the prev vout extra data?
func (pgb *ChainDBRPC) InsightGetAddressTransactions(addr string, count,
	skip int) []*dcrjson.SearchRawTransactionsResult {
	address, err := exccutil.DecodeAddress(addr)
	if err != nil {
		log.Infof("Invalid address %s: %v", addr, err)
		return nil
	}
	prevVoutExtraData := true
	txs, err := pgb.Client.SearchRawTransactionsVerbose(
		address, skip, count, prevVoutExtraData, true, nil)

	if err != nil {
		log.Warnf("GetAddressTransactions failed for address %s: %v", addr, err)
		return nil
	}
	return txs
}

// GetTransactionHex returns the full serialized transaction for the specified
// transaction hash as a hex encode string.
func (pgb *ChainDBRPC) GetTransactionHex(txid string) string {
	txraw, err := rpcutils.GetTransactionVerboseByID(pgb.Client, txid)

	if err != nil {
		log.Errorf("GetRawTransactionVerbose failed for: %v", err)
		return ""
	}

	return txraw.Hex
}

// GetBlockVerboseByHash returns a *dcrjson.GetBlockVerboseResult for the
// specified block hash, optionally with transaction details.
func (pgb *ChainDBRPC) GetBlockVerboseByHash(hash string, verboseTx bool) *dcrjson.GetBlockVerboseResult {
	return rpcutils.GetBlockVerboseByHash(pgb.Client, pgb.ChainDB.chainParams,
		hash, verboseTx)
}

// GetTransactionsForBlockByHash returns a *apitypes.BlockTransactions for the
// block with the specified hash.
func (pgb *ChainDBRPC) GetTransactionsForBlockByHash(hash string) *apitypes.BlockTransactions {
	blockVerbose := rpcutils.GetBlockVerboseByHash(
		pgb.Client, pgb.ChainDB.chainParams, hash, false)

	return makeBlockTransactions(blockVerbose)
}

func makeBlockTransactions(blockVerbose *dcrjson.GetBlockVerboseResult) *apitypes.BlockTransactions {
	blockTransactions := new(apitypes.BlockTransactions)

	blockTransactions.Tx = make([]string, len(blockVerbose.Tx))
	copy(blockTransactions.Tx, blockVerbose.Tx)

	blockTransactions.STx = make([]string, len(blockVerbose.STx))
	copy(blockTransactions.STx, blockVerbose.STx)

	return blockTransactions
}

// GetBlockHash returns the hash of the block at the specified height.
func (pgb *ChainDB) GetBlockHash(idx int64) (string, error) {
	hash, err := RetrieveBlockHash(pgb.db, idx)
	if err != nil {
		log.Errorf("Unable to get block hash for block number %d: %v", idx, err)
		return "", err
	}
	return hash, nil
}

// GetAddressBalance returns a *explorer.AddressBalance for the specified
// address, transaction count limit, and transaction number offset.
func (pgb *ChainDB) GetAddressBalance(address string, N, offset int64) *explorer.AddressBalance {
	_, balance, err := pgb.AddressHistoryAll(address, N, offset)
	if err != nil {
		return nil
	}
	return balance
}

// GetAddressInfo returns the basic information for the specified address
// (*apitypes.InsightAddressInfo), given a transaction count limit, and
// transaction number offset.
func (pgb *ChainDB) GetAddressInfo(address string, N, offset int64) *apitypes.InsightAddressInfo {
	rows, balance, err := pgb.AddressHistoryAll(address, N, offset)
	if err != nil {
		return nil
	}

	var totalReceived, totalSent, unSpent exccutil.Amount
	totalReceived, _ = exccutil.NewAmount(float64(balance.TotalSpent + balance.TotalUnspent))
	totalSent, _ = exccutil.NewAmount(float64(balance.TotalSpent))
	unSpent, _ = exccutil.NewAmount(float64(balance.TotalUnspent))

	var transactionIdList []string
	for _, row := range rows {
		fundingTxId := row.FundingTxHash
		if fundingTxId != "" {
			transactionIdList = append(transactionIdList, fundingTxId)
		}

		spendingTxId := row.SpendingTxHash
		if spendingTxId != "" {
			transactionIdList = append(transactionIdList, spendingTxId)
		}
	}

	return &apitypes.InsightAddressInfo{
		Address:        address,
		TotalReceived:  totalReceived,
		TransactionsID: transactionIdList,
		TotalSent:      totalSent,
		Unspent:        unSpent,
	}
}

// GetBlockSummaryTimeRange returns the blocks created within a specified time
// range (min, max time), up to limit transactions.
func (pgb *ChainDB) GetBlockSummaryTimeRange(min, max int64, limit int) []dbtypes.BlockDataBasic {
	blockSummary, err := RetrieveBlockSummaryByTimeRange(pgb.db, min, max, limit)
	if err != nil {
		log.Errorf("Unable to retrieve block summary using time %d: %v", min, err)
	}
	return blockSummary
}

// GetAddressUTXO returns the unspent transaction outputs (UTXOs) paying to the
// specified address in a []apitypes.AddressTxnOutput.
func (pgb *ChainDB) GetAddressUTXO(address string) []apitypes.AddressTxnOutput {
	blockHeight, _, _, err := RetrieveBestBlockHeight(pgb.db)
	if err != nil {
		log.Error(err)
		return nil
	}
	txnOutput, err := RetrieveAddressTxnOutputWithTransaction(pgb.db, address, int64(blockHeight))
	if err != nil {
		log.Error(err)
		return nil
	}
	return txnOutput
}

// GetAddressSpendByFunHash will return the address that fundex a tx
func (pgb *ChainDB) GetAddressSpendByFunHash(addresses []string, fundHash string) []*apitypes.AddressSpendByFunHash {

	AddrRow, err := RetrieveAddressTxnsByFundingTx(pgb.db, fundHash, addresses)
	if err != nil {
		log.Error(err)
		return nil
	}
	return AddrRow
}
