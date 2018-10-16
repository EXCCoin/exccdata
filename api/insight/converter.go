// Copyright (c) 2018 The ExchangeCoin team
// Copyright (c) 2018, The Decred developers
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package insight

import (
	"github.com/EXCCoin/exccd/blockchain"
	"github.com/EXCCoin/exccd/exccjson"
	"github.com/EXCCoin/exccd/exccutil"
	apitypes "github.com/EXCCoin/exccdata/v3/api/types"
)

// TxConverter converts exccd-tx to insight tx
func (c *insightApiContext) TxConverter(txs []*exccjson.TxRawResult) ([]apitypes.InsightTx, error) {
	return c.ExccToInsightTxns(txs, false, false, false)
}

// ExccToInsightTxns takes struct with filter params
func (c *insightApiContext) ExccToInsightTxns(txs []*exccjson.TxRawResult,
	noAsm, noScriptSig, noSpent bool) ([]apitypes.InsightTx, error) {
	var newTxs []apitypes.InsightTx
	for _, tx := range txs {

		// Build new InsightTx
		txNew := apitypes.InsightTx{
			Txid:          tx.Txid,
			Version:       tx.Version,
			Locktime:      tx.LockTime,
			Blockhash:     tx.BlockHash,
			Blockheight:   tx.BlockHeight,
			Confirmations: tx.Confirmations,
			Time:          tx.Time,
			Blocktime:     tx.Blocktime,
			Size:          uint32(len(tx.Hex) / 2),
		}

		// Vins fill
		var vInSum, vOutSum float64

		for vinID, vin := range tx.Vin {

			InsightVin := &apitypes.InsightVin{
				Txid:     vin.Txid,
				Vout:     vin.Vout,
				Sequence: vin.Sequence,
				N:        vinID,
				Value:    vin.AmountIn,
				CoinBase: vin.Coinbase,
			}

			// init ScriptPubKey
			if !noScriptSig {
				InsightVin.ScriptSig = new(apitypes.InsightScriptSig)
				if vin.ScriptSig != nil {
					if !noAsm {
						InsightVin.ScriptSig.Asm = vin.ScriptSig.Asm
					}
					InsightVin.ScriptSig.Hex = vin.ScriptSig.Hex
				}
			}

			// Note, this only gathers information from the database which does not include mempool transactions
			_, addresses, value, err := c.BlockData.ChainDB.RetrieveAddressIDsByOutpoint(vin.Txid, vin.Vout)
			if err == nil {
				if len(addresses) > 0 {
					// Update Vin due to EXCCD AMOUNTIN - START
					// NOTE THIS IS ONLY USEFUL FOR INPUT AMOUNTS THAT ARE NOT ALSO FROM MEMPOOL
					if tx.Confirmations == 0 {
						InsightVin.Value = exccutil.Amount(value).ToCoin()
					}
					// Update Vin due to EXCCD AMOUNTIN - END
					InsightVin.Addr = addresses[0]
				}
			}
			exccamt, _ := exccutil.NewAmount(InsightVin.Value)
			InsightVin.ValueSat = int64(exccamt)

			vInSum += InsightVin.Value
			txNew.Vins = append(txNew.Vins, InsightVin)

		}

		// Vout fill
		for _, v := range tx.Vout {
			InsightVout := &apitypes.InsightVout{
				Value: v.Value,
				N:     v.N,
				ScriptPubKey: apitypes.InsightScriptPubKey{
					Addresses: v.ScriptPubKey.Addresses,
					Type:      v.ScriptPubKey.Type,
					Hex:       v.ScriptPubKey.Hex,
				},
			}
			if !noAsm {
				InsightVout.ScriptPubKey.Asm = v.ScriptPubKey.Asm
			}

			txNew.Vouts = append(txNew.Vouts, InsightVout)
			vOutSum += v.Value
		}

		exccamt, _ := exccutil.NewAmount(vOutSum)
		txNew.ValueOut = exccamt.ToCoin()

		exccamt, _ = exccutil.NewAmount(vInSum)
		txNew.ValueIn = exccamt.ToCoin()

		exccamt, _ = exccutil.NewAmount(txNew.ValueIn - txNew.ValueOut)
		txNew.Fees = exccamt.ToCoin()

		// Return true if coinbase value is not empty, return 0 at some fields
		if txNew.Vins != nil && txNew.Vins[0].CoinBase != "" {
			txNew.IsCoinBase = true
			txNew.ValueIn = 0
			txNew.Fees = 0
			for _, v := range txNew.Vins {
				v.Value = 0
				v.ValueSat = 0
			}
		}

		if !noSpent {
			// populate the spending status of all vouts
			// Note, this only gathers information from the database which does not include mempool transactions
			addrFull := c.BlockData.ChainDB.GetSpendDetailsByFundingHash(txNew.Txid)
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

// ExccToInsightBlock converts a exccjson.GetBlockVerboseResult to Insight block.
func (c *insightApiContext) ExccToInsightBlock(inBlocks []*exccjson.GetBlockVerboseResult) ([]*apitypes.InsightBlockResult, error) {
	RewardAtBlock := func(blocknum int64, voters uint16) float64 {
		subsidyCache := blockchain.NewSubsidyCache(0, c.params)
		work := blockchain.CalcBlockWorkSubsidy(subsidyCache, blocknum, voters, c.params)
		stake := blockchain.CalcStakeVoteSubsidy(subsidyCache, blocknum, c.params) * int64(voters)
		tax := blockchain.CalcBlockTaxSubsidy(subsidyCache, blocknum, voters, c.params)
		return exccutil.Amount(work + stake + tax).ToCoin()
	}

	outBlocks := make([]*apitypes.InsightBlockResult, 0, len(inBlocks))
	for _, inBlock := range inBlocks {
		outBlock := apitypes.InsightBlockResult{
			Hash:          inBlock.Hash,
			Confirmations: inBlock.Confirmations,
			Size:          inBlock.Size,
			Height:        inBlock.Height,
			Version:       inBlock.Version,
			MerkleRoot:    inBlock.MerkleRoot,
			Tx:            append(inBlock.Tx, inBlock.STx...),
			Time:          inBlock.Time,
			Nonce:         inBlock.Nonce,
			Bits:          inBlock.Bits,
			Difficulty:    inBlock.Difficulty,
			PreviousHash:  inBlock.PreviousHash,
			NextHash:      inBlock.NextHash,
			Reward:        RewardAtBlock(inBlock.Height, inBlock.Voters),
			IsMainChain:   inBlock.Height > 0,
		}
		outBlocks = append(outBlocks, &outBlock)
	}
	return outBlocks, nil
}
