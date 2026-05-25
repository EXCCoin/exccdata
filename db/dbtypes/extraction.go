package dbtypes

import (
	"fmt"

	"github.com/EXCCoin/exccd/blockchain/stake/v4"
	"github.com/EXCCoin/exccd/chaincfg/v3"
	"github.com/EXCCoin/exccd/txscript/v4/stdscript"
	"github.com/EXCCoin/exccd/wire"
	"github.com/EXCCoin/exccdata/v8/txhelpers"
)

func ExtractBlockTransactions(msgBlock *wire.MsgBlock, txTree int8,
	chainParams *chaincfg.Params, isValid, isMainchain bool) ([]*Tx, [][]*Vout, []VinTxPropertyARRAY) {
	dbTxs, dbTxVouts, dbTxVins := processTransactions(msgBlock, txTree,
		chainParams, isValid, isMainchain)
	if txTree != wire.TxTreeRegular && txTree != wire.TxTreeStake {
		fmt.Printf("Invalid transaction tree: %v", txTree)
	}
	return dbTxs, dbTxVouts, dbTxVins
}

func processTransactions(msgBlock *wire.MsgBlock, tree int8, chainParams *chaincfg.Params,
	isValid, isMainchain bool) ([]*Tx, [][]*Vout, []VinTxPropertyARRAY) {

	var stakeTree bool
	var txs []*wire.MsgTx
	switch tree {
	case wire.TxTreeRegular:
		txs = msgBlock.Transactions
	case wire.TxTreeStake:
		txs = msgBlock.STransactions
		stakeTree = true
	default:
		return nil, nil, nil
	}

	blockHeight := msgBlock.Header.Height
	blockHash := ChainHash(msgBlock.BlockHash())
	blockTime := NewTimeDef(msgBlock.Header.Timestamp)

	treasuryActive := stakeTree && txhelpers.IsTreasuryActive(chainParams.Net, int64(blockHeight))

	dbTransactions := make([]*Tx, 0, len(txs))
	dbTxVouts := make([][]*Vout, len(txs))
	dbTxVins := make([]VinTxPropertyARRAY, len(txs))

	ticketPrice := msgBlock.Header.SBits

	for txIndex, tx := range txs {
		txType := txhelpers.DetermineTxType(tx, treasuryActive)
		isStake := txType != stake.TxTypeRegular
		if isStake && !stakeTree {
			fmt.Printf(" ***************** INCONSISTENT TREE: txn %v, type = %v", tx.TxHash(), txType)
			continue
		}

		var mixDenom int64
		var mixCount uint32
		if !isStake {
			_, mixDenom, mixCount = txhelpers.IsMixTx(tx)
			if mixCount == 0 {
				_, mixDenom, mixCount = txhelpers.IsMixedSplitTx(tx, int64(txhelpers.DefaultRelayFeePerKb), ticketPrice)
			}
		}
		var spent, sent int64
		for _, txin := range tx.TxIn {
			spent += txin.ValueIn
		}
		for _, txout := range tx.TxOut {
			sent += txout.Value
		}
		fees := spent - sent
		dbTx := &Tx{
			BlockHash:        blockHash,
			BlockHeight:      int64(blockHeight),
			BlockTime:        blockTime,
			TxType:           int16(txType),
			Version:          tx.Version,
			Tree:             tree,
			TxID:             ChainHash(*tx.CachedTxHash()),
			BlockIndex:       uint32(txIndex),
			Locktime:         tx.LockTime,
			Expiry:           tx.Expiry,
			Size:             uint32(tx.SerializeSize()),
			Spent:            spent,
			Sent:             sent,
			Fees:             fees,
			MixCount:         int32(mixCount),
			MixDenom:         mixDenom,
			NumVin:           uint32(len(tx.TxIn)),
			NumVout:          uint32(len(tx.TxOut)),
			IsValid:          isValid || tree == wire.TxTreeStake,
			IsMainchainBlock: isMainchain,
		}

		dbTxVins[txIndex] = make(VinTxPropertyARRAY, 0, len(tx.TxIn))
		for idx, txin := range tx.TxIn {
			dbTxVins[txIndex] = append(dbTxVins[txIndex], VinTxProperty{
				PrevTxHash:  ChainHash(txin.PreviousOutPoint.Hash),
				PrevTxIndex: txin.PreviousOutPoint.Index,
				PrevTxTree:  uint16(txin.PreviousOutPoint.Tree),
				Sequence:    txin.Sequence,
				ValueIn:     txin.ValueIn,
				TxID:        dbTx.TxID,
				TxIndex:     uint32(idx),
				TxType:      dbTx.TxType,
				TxTree:      uint16(dbTx.Tree),
				Time:        blockTime,
				BlockHeight: txin.BlockHeight,
				BlockIndex:  txin.BlockIndex,
				ScriptSig:   txin.SignatureScript,
				IsValid:     dbTx.IsValid,
				IsMainchain: isMainchain,
			})
		}

		dbTxVouts[txIndex] = make([]*Vout, 0, len(tx.TxOut))
		for io, txout := range tx.TxOut {
			vout := Vout{
				TxHash:  dbTx.TxID,
				TxIndex: uint32(io),
				TxTree:  tree,
				TxType:  dbTx.TxType,
				Value:   uint64(txout.Value),
				Version: txout.Version,
				Mixed:   mixDenom > 0 && mixDenom == txout.Value,
			}
			scriptClass, scriptAddrs := stdscript.ExtractAddrs(vout.Version, txout.PkScript, chainParams)
			addys := make([]string, 0, len(scriptAddrs))
			for ia := range scriptAddrs {
				addys = append(addys, scriptAddrs[ia].String())
			}
			vout.ScriptPubKeyData.Type = NewScriptClass(scriptClass)
			vout.ScriptPubKeyData.Addresses = addys
			dbTxVouts[txIndex] = append(dbTxVouts[txIndex], &vout)
		}

		dbTransactions = append(dbTransactions, dbTx)
	}

	return dbTransactions, dbTxVouts, dbTxVins
}
