// Copyright (c) 2018, The Decred developers
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package insight

import (
	"github.com/EXCCoin/exccd/exccjson"
	"github.com/EXCCoin/exccd/exccutil"
	apitypes "github.com/EXCCoin/exccdata/api/types"
)

// TxConverter converts exccd-tx to insight tx
func (c *insightApiContext) TxConverter(txs []*exccjson.TxRawResult) ([]apitypes.InsightTx, error) {
	return c.TxConverterWithParams(txs, false, false, false)
}

// TxConverterWithParams takes struct with filter params
func (c *insightApiContext) TxConverterWithParams(txs []*exccjson.TxRawResult, noAsm bool, noScriptSig bool, noSpent bool) ([]apitypes.InsightTx, error) {
	newTxs := []apitypes.InsightTx{}
	for _, tx := range txs {

		vInSum := float64(0)
		vOutSum := float64(0)

		// Build new model. Based on the old api responses of
		txNew := apitypes.InsightTx{}
		txNew.Txid = tx.Txid
		txNew.Version = tx.Version
		txNew.Locktime = tx.LockTime

		// Vins fill
		for vinID, vin := range tx.Vin {
			vinEmpty := &apitypes.InsightVin{}
			emptySS := &apitypes.InsightScriptSig{}
			txNew.Vins = append(txNew.Vins, vinEmpty)
			txNew.Vins[vinID].Txid = vin.Txid
			txNew.Vins[vinID].Vout = vin.Vout
			txNew.Vins[vinID].Sequence = vin.Sequence

			txNew.Vins[vinID].CoinBase = vin.Coinbase

			// init ScriptPubKey
			if !noScriptSig {
				txNew.Vins[vinID].ScriptSig = emptySS
				if vin.ScriptSig != nil {
					if !noAsm {
						txNew.Vins[vinID].ScriptSig.Asm = vin.ScriptSig.Asm
					}
					txNew.Vins[vinID].ScriptSig.Hex = vin.ScriptSig.Hex
				}
			}

			txNew.Vins[vinID].N = vinID

			txNew.Vins[vinID].Value = vin.AmountIn

			// Lookup addresses OPTION 2
			// Note, this only gathers information from the database which does not include mempool transactions
			_, addresses, value, err := c.BlockData.ChainDB.RetrieveAddressIDsByOutpoint(vin.Txid, vin.Vout)
			if err == nil {
				if len(addresses) > 0 {
					// Update Vin due to EXCCD AMOUNTIN - START
					// NOTE THIS IS ONLY USEFUL FOR INPUT AMOUNTS THAT ARE NOT ALSO FROM MEMPOOL
					if tx.Confirmations == 0 {
						txNew.Vins[vinID].Value = exccutil.Amount(value).ToCoin()
					}
					// Update Vin due to EXCCD AMOUNTIN - END
					txNew.Vins[vinID].Addr = addresses[0]
				}
			}
			exccamt, _ := exccutil.NewAmount(txNew.Vins[vinID].Value)
			txNew.Vins[vinID].ValueSat = int64(exccamt)
			vInSum += txNew.Vins[vinID].Value

		}

		// Vout fill
		for _, v := range tx.Vout {
			voutEmpty := &apitypes.InsightVout{}
			emptyPubKey := apitypes.InsightScriptPubKey{}
			txNew.Vouts = append(txNew.Vouts, voutEmpty)
			txNew.Vouts[v.N].Value = v.Value
			vOutSum += v.Value
			txNew.Vouts[v.N].N = v.N
			// pk block
			txNew.Vouts[v.N].ScriptPubKey = emptyPubKey
			if !noAsm {
				txNew.Vouts[v.N].ScriptPubKey.Asm = v.ScriptPubKey.Asm
			}
			txNew.Vouts[v.N].ScriptPubKey.Hex = v.ScriptPubKey.Hex
			txNew.Vouts[v.N].ScriptPubKey.Type = v.ScriptPubKey.Type
			txNew.Vouts[v.N].ScriptPubKey.Addresses = v.ScriptPubKey.Addresses
		}

		txNew.Blockhash = tx.BlockHash
		txNew.Blockheight = tx.BlockHeight
		txNew.Confirmations = tx.Confirmations
		txNew.Time = tx.Time
		txNew.Blocktime = tx.Blocktime
		txNew.Size = uint32(len(tx.Hex) / 2)

		exccamt, _ := exccutil.NewAmount(vOutSum)
		txNew.ValueOut = exccamt.ToCoin()

		exccamt, _ = exccutil.NewAmount(vInSum)
		txNew.ValueIn = exccamt.ToCoin()

		exccamt, _ = exccutil.NewAmount(txNew.ValueIn - txNew.ValueOut)
		txNew.Fees = exccamt.ToCoin()

		// Return true if coinbase value is not empty, return 0 at some fields
		if txNew.Vins != nil && len(txNew.Vins[0].CoinBase) > 0 {
			txNew.IsCoinBase = true
			txNew.ValueIn = 0
			txNew.Fees = 0
			for _, v := range txNew.Vins {
				v.Value = 0
				v.ValueSat = 0
			}
		}

		if !noSpent {

			// set of unique addresses for db query
			uniqAddrs := make(map[string]string)

			for _, vout := range txNew.Vouts {
				for _, addr := range vout.ScriptPubKey.Addresses {
					uniqAddrs[addr] = txNew.Txid
				}
			}

			addresses := []string{}
			for addr := range uniqAddrs {
				addresses = append(addresses, addr)
			}
			// Note, this only gathers information from the database which does not include mempool transactions
			addrFull := c.BlockData.ChainDB.GetAddressSpendByFunHash(addresses, txNew.Txid)
			for _, dbaddr := range addrFull {
				txNew.Vouts[dbaddr.FundingTxVoutIndex].SpentIndex = dbaddr.SpendingTxVinIndex
				txNew.Vouts[dbaddr.FundingTxVoutIndex].SpentTxID = dbaddr.SpendingTxHash
				txNew.Vouts[dbaddr.FundingTxVoutIndex].SpentHeight = dbaddr.BlockHeight
			}
		}
		newTxs = append(newTxs, txNew)
	}
	return newTxs, nil
}
