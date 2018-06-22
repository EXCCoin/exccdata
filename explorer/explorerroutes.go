// Copyright (c) 2018, The Decred developers
// Copyright (c) 2017, The dcrdata developers
// See LICENSE for details.

package explorer

import (
	"database/sql"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/EXCCoin/exccd/chaincfg"
	"github.com/EXCCoin/exccd/chaincfg/chainhash"
	"github.com/EXCCoin/exccd/exccutil"
	"github.com/EXCCoin/exccd/wire"
	"github.com/EXCCoin/exccdata/db/dbtypes"
	"github.com/EXCCoin/exccdata/txhelpers"
	humanize "github.com/dustin/go-humanize"
)

// netName returns the name used when referring to a excc network.
func netName(chainParams *chaincfg.Params) string {
	switch chainParams.Net {
	case wire.TestNet2:
		return "Testnet"
	default:
		return strings.Title(chainParams.Name)
	}
}

// Home is the page handler for the "/" path
func (exp *explorerUI) Home(w http.ResponseWriter, r *http.Request) {
	height := exp.blockData.GetHeight()

	end := height - 5
	if end < 0 {
		end = 0
	}
	
	blocks := exp.blockData.GetExplorerBlocks(height, end)

	exp.NewBlockDataMtx.Lock()
	exp.MempoolData.RLock()

	str, err := exp.templates.execTemplateToString("home", struct {
		Info    *HomeInfo
		Mempool *MempoolInfo
		Blocks  []*BlockBasic
		Version string
		NetName string
	}{
		exp.ExtraInfo,
		exp.MempoolData,
		blocks,
		exp.Version,
		exp.NetName,
	})
	exp.NewBlockDataMtx.Unlock()
	exp.MempoolData.RUnlock()

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// Blocks is the page handler for the "/blocks" path
func (exp *explorerUI) Blocks(w http.ResponseWriter, r *http.Request) {
	idx := exp.blockData.GetHeight()

	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil || height > idx {
		height = idx
	}

	rows, err := strconv.Atoi(r.URL.Query().Get("rows"))

	if err != nil || rows > maxExplorerRows || rows < minExplorerRows {
		rows = minExplorerRows
	}

	oldestBlock := height - rows + 1
	if oldestBlock < 0 {
		height = rows - 1
	}

	summaries := exp.blockData.GetExplorerBlocks(height, height-rows)
	if summaries == nil {
		log.Errorf("Unable to get blocks: height=%d&rows=%d", height, rows)
		exp.ErrorPage(w, "Something went wrong...", "could not find those blocks", true)
		return
	}

	str, err := exp.templates.execTemplateToString("explorer", struct {
		Data      []*BlockBasic
		BestBlock int
		Rows      int
		Version   string
		NetName   string
	}{
		summaries,
		idx,
		rows,
		exp.Version,
		exp.NetName,
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// Block is the page handler for the "/block" path
func (exp *explorerUI) Block(w http.ResponseWriter, r *http.Request) {
	hash := getBlockHashCtx(r)

	data := exp.blockData.GetExplorerBlock(hash)
	if data == nil {
		log.Errorf("Unable to get block %s", hash)
		exp.ErrorPage(w, "Something went wrong...", "could not find that block", true)
		return
	}
	// Checking if there exists any regular non-Coinbase transactions in the block.
	var count int
	data.TxAvailable = true
	for _, i := range data.Tx {
		if i.Coinbase {
			count++
		}
	}
	if count == len(data.Tx) {
		data.TxAvailable = false
	}

	if !exp.liteMode {
		var err error
		data.Misses, err = exp.explorerSource.BlockMissedVotes(hash)
		if err != nil && err != sql.ErrNoRows {
			log.Warnf("Unable to retrieve missed votes for block %s: %v", hash, err)
		}
	}

	pageData := struct {
		Data          *BlockInfo
		ConfirmHeight int64
		Version       string
		NetName       string
	}{
		data,
		exp.NewBlockData.Height - data.Confirmations,
		exp.Version,
		exp.NetName,
	}
	str, err := exp.templates.execTemplateToString("block", pageData)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Turbolinks-Location", r.URL.RequestURI())
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// Mempool is the page handler for the "/mempool" path
func (exp *explorerUI) Mempool(w http.ResponseWriter, r *http.Request) {
	exp.MempoolData.RLock()
	str, err := exp.templates.execTemplateToString("mempool", struct {
		Mempool *MempoolInfo
		Version string
		NetName string
	}{
		exp.MempoolData,
		exp.Version,
		exp.NetName,
	})
	exp.MempoolData.RUnlock()

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// TxPage is the page handler for the "/tx" path
func (exp *explorerUI) TxPage(w http.ResponseWriter, r *http.Request) {
	// attempt to get tx hash string from URL path
	hash, ok := r.Context().Value(ctxTxHash).(string)
	if !ok {
		log.Trace("txid not set")
		exp.ErrorPage(w, "Something went wrong...", "there was no transaction requested", true)
		return
	}
	tx := exp.blockData.GetExplorerTx(hash)
	if tx == nil {
		log.Errorf("Unable to get transaction %s", hash)
		exp.ErrorPage(w, "Something went wrong...", "could not find that transaction", true)
		return
	}
	if !exp.liteMode {
		// For any coinbase transactions look up the total block fees to include as part of the inputs
		if tx.Type == "Coinbase" {
			data := exp.blockData.GetExplorerBlock(tx.BlockHash)
			if data == nil {
				log.Errorf("Unable to get block %s", tx.BlockHash)
			} else {
				tx.BlockMiningFee = int64(data.MiningFee)
			}
		}
		// For each output of this transaction, look up any spending transactions,
		// and the index of the spending transaction input.
		spendingTxHashes, spendingTxVinInds, voutInds, err := exp.explorerSource.SpendingTransactions(hash)
		if err != nil {
			log.Errorf("Unable to retrieve spending transactions for %s: %v", hash, err)
			exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
			return
		}
		for i, vout := range voutInds {
			if int(vout) >= len(tx.SpendingTxns) {
				log.Errorf("Invalid spending transaction data (%s:%d)", hash, vout)
				continue
			}
			tx.SpendingTxns[vout] = TxInID{
				Hash:  spendingTxHashes[i],
				Index: spendingTxVinInds[i],
			}
		}
		if tx.Type == "Ticket" {
			spendStatus, poolStatus, err := exp.explorerSource.PoolStatusForTicket(hash)
			if err != nil {
				log.Errorf("Unable to retrieve ticket spend and pool status for %s: %v", hash, err)
			} else {
				if tx.Mature == "False" {
					tx.TicketInfo.PoolStatus = "immature"
				} else {
					tx.TicketInfo.PoolStatus = poolStatus.String()
				}
				tx.TicketInfo.SpendStatus = spendStatus.String()
				blocksLive := tx.Confirmations - int64(exp.ChainParams.TicketMaturity)
				tx.TicketInfo.TicketPoolSize = int64(exp.ChainParams.TicketPoolSize) * int64(exp.ChainParams.TicketsPerBlock)
				tx.TicketInfo.TicketExpiry = int64(exp.ChainParams.TicketExpiry)
				expirationInDays := (exp.ChainParams.TargetTimePerBlock.Hours() * float64(exp.ChainParams.TicketExpiry)) / 24
				maturityInDay := (exp.ChainParams.TargetTimePerBlock.Hours() * float64(tx.TicketInfo.TicketMaturity)) / 24
				tx.TicketInfo.TimeTillMaturity = ((float64(exp.ChainParams.TicketMaturity) -
					float64(tx.Confirmations)) / float64(exp.ChainParams.TicketMaturity)) * maturityInDay
				ticketExpiryBlocksLeft := int64(exp.ChainParams.TicketExpiry) - blocksLive
				tx.TicketInfo.TicketExpiryDaysLeft = (float64(ticketExpiryBlocksLeft) /
					float64(exp.ChainParams.TicketExpiry)) * expirationInDays
				if tx.TicketInfo.SpendStatus == "Voted" {
					// Blocks from eligible until voted (actual luck)
					tx.TicketInfo.TicketLiveBlocks = exp.blockData.TxHeight(tx.SpendingTxns[0].Hash) -
						tx.BlockHeight - int64(exp.ChainParams.TicketMaturity) - 1
				} else if tx.Confirmations >= int64(exp.ChainParams.TicketExpiry+
					uint32(exp.ChainParams.TicketMaturity)) { // Expired
					// Blocks ticket was active before expiring (actual no luck)
					tx.TicketInfo.TicketLiveBlocks = int64(exp.ChainParams.TicketExpiry)
				} else { // Active
					// Blocks ticket has been active and eligible to vote
					tx.TicketInfo.TicketLiveBlocks = blocksLive
				}
				tx.TicketInfo.BestLuck = tx.TicketInfo.TicketExpiry / int64(exp.ChainParams.TicketPoolSize)
				tx.TicketInfo.AvgLuck = tx.TicketInfo.BestLuck - 1
				if tx.TicketInfo.TicketLiveBlocks == int64(exp.ChainParams.TicketExpiry) {
					tx.TicketInfo.VoteLuck = 0
				} else {
					tx.TicketInfo.VoteLuck = float64(tx.TicketInfo.BestLuck) -
						(float64(tx.TicketInfo.TicketLiveBlocks) / float64(exp.ChainParams.TicketPoolSize))
				}
				if tx.TicketInfo.VoteLuck >= float64(tx.TicketInfo.BestLuck-(1/int64(exp.ChainParams.TicketPoolSize))) {
					tx.TicketInfo.LuckStatus = "Perfection"
				} else if tx.TicketInfo.VoteLuck > (float64(tx.TicketInfo.BestLuck) - 0.25) {
					tx.TicketInfo.LuckStatus = "Very Lucky!"
				} else if tx.TicketInfo.VoteLuck > (float64(tx.TicketInfo.BestLuck) - 0.75) {
					tx.TicketInfo.LuckStatus = "Good Luck"
				} else if tx.TicketInfo.VoteLuck > (float64(tx.TicketInfo.BestLuck) - 1.25) {
					tx.TicketInfo.LuckStatus = "Normal"
				} else if tx.TicketInfo.VoteLuck > (float64(tx.TicketInfo.BestLuck) * 0.50) {
					tx.TicketInfo.LuckStatus = "Bad Luck"
				} else if tx.TicketInfo.VoteLuck > 0 {
					tx.TicketInfo.LuckStatus = "Horrible Luck!"
				} else if tx.TicketInfo.VoteLuck == 0 {
					tx.TicketInfo.LuckStatus = "No Luck"
				}

				// Chance for a ticket to NOT be voted in a given time frame:
				// C = (1 - P)^N
				// Where: P is the probability of a vote in one block. (votes
				// per block / current ticket pool size)
				// N is the number of blocks before ticket expiry. (ticket
				// expiry in blocks - (number of blocks since ticket purchase -
				// ticket maturity))
				// C is the probability (chance)
				pVote := float64(exp.ChainParams.TicketsPerBlock) / float64(exp.ExtraInfo.PoolInfo.Size)
				tx.TicketInfo.Probability = 100 * (math.Pow(1-pVote,
					float64(exp.ChainParams.TicketExpiry)-float64(blocksLive)))
			}
		}
	}

	pageData := struct {
		Data          *TxInfo
		ConfirmHeight int64
		Version       string
		NetName       string
	}{
		tx,
		exp.NewBlockData.Height - tx.Confirmations,
		exp.Version,
		exp.NetName,
	}

	str, err := exp.templates.execTemplateToString("tx", pageData)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Turbolinks-Location", r.URL.RequestURI())
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// AddressPage is the page handler for the "/address" path
func (exp *explorerUI) AddressPage(w http.ResponseWriter, r *http.Request) {
	// AddressPageData is the data structure passed to the HTML template
	type AddressPageData struct {
		Data          *AddressInfo
		ConfirmHeight []int64
		Version       string
		NetName       string
	}

	// Get the address URL parameter, which should be set in the request context
	// by the addressPathCtx middleware.
	address, ok := r.Context().Value(ctxAddress).(string)
	if !ok {
		log.Trace("address not set")
		exp.ErrorPage(w, "Something went wrong...", "there seems to not be an address in this request", true)
		return
	}

	// Number of outputs for the address to query the database for. The URL
	// query parameter "n" is used to specify the limit (e.g. "?n=20").
	limitN, err := strconv.ParseInt(r.URL.Query().Get("n"), 10, 64)
	if err != nil || limitN < 0 {
		limitN = defaultAddressRows
	} else if limitN > MaxAddressRows {
		log.Warnf("addressPage: requested up to %d address rows, "+
			"limiting to %d", limitN, MaxAddressRows)
		limitN = MaxAddressRows
	}

	// Number of outputs to skip (OFFSET in database query). For UX reasons, the
	// "start" URL query parameter is used.
	offsetAddrOuts, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	if err != nil || offsetAddrOuts < 0 {
		offsetAddrOuts = 0
	}

	// Transaction types to show.
	txntype := r.URL.Query().Get("txntype")
	if txntype == "" {
		txntype = "all"
	}
	txnType := dbtypes.AddrTxnTypeFromStr(txntype)
	if txnType == dbtypes.AddrTxnUnknown {
		exp.ErrorPage(w, "Something went wrong...", "unknown txntype query value", false)
		return
	}
	log.Debugf("Showing transaction types: %s (%d)", txntype, txnType)

	// Retrieve address information from the DB and/or RPC
	var addrData *AddressInfo
	if exp.liteMode {
		addrData = exp.blockData.GetExplorerAddress(address, limitN, offsetAddrOuts)
		if addrData == nil {
			log.Errorf("Unable to get address %s", address)
			exp.ErrorPage(w, "Something went wrong...", "could not find that address", true)
			return
		}
	} else {
		// Get addresses table rows for the address
		addrHist, balance, errH := exp.explorerSource.AddressHistory(
			address, limitN, offsetAddrOuts, txnType)
		// Fallback to RPC if DB query fails
		if errH != nil {
			log.Errorf("Unable to get address %s history: %v", address, errH)
			addrData = exp.blockData.GetExplorerAddress(address, limitN, offsetAddrOuts)
			if addrData == nil {
				log.Errorf("Unable to get address %s", address)
				exp.ErrorPage(w, "Something went wrong...", "could not find that address", true)
				return
			}

			// Set page parameters
			addrData.Path = r.URL.Path
			addrData.Limit, addrData.Offset = limitN, offsetAddrOuts
			addrData.TxnType = txnType.String()

			confirmHeights := make([]int64, len(addrData.Transactions))
			for i, v := range addrData.Transactions {
				confirmHeights[i] = exp.NewBlockData.Height - int64(v.Confirmations)
			}

			pageData := AddressPageData{
				Data:          addrData,
				ConfirmHeight: confirmHeights,
				Version:       exp.Version,
			}
			str, err := exp.templates.execTemplateToString("address", pageData)
			if err != nil {
				log.Errorf("Template execute failure: %v", err)
				exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, str)
			return
		}

		// Generate AddressInfo skeleton from the address table rows
		addrData = ReduceAddressHistory(addrHist)
		if addrData == nil {
			// Empty history is not expected for credit txnType with any txns.
			if txnType != dbtypes.AddrTxnDebit && (balance.NumSpent+balance.NumUnspent) > 0 {
				log.Debugf("empty address history (%s): n=%d&start=%d", address, limitN, offsetAddrOuts)
				exp.ErrorPage(w, "Something went wrong...", "that address has no history", true)
				return
			}
			// No mined transactions
			addrData = new(AddressInfo)
			addrData.Address = address
		}
		addrData.Fullmode = true

		// Balances and txn counts (partial unless in full mode)
		addrData.Balance = balance
		addrData.KnownTransactions = (balance.NumSpent * 2) + balance.NumUnspent
		addrData.KnownFundingTxns = balance.NumSpent + balance.NumUnspent
		addrData.KnownSpendingTxns = balance.NumSpent

		// Transactions to fetch with FillAddressTransactions. This should be a
		// noop if ReduceAddressHistory is working right.
		switch txnType {
		case dbtypes.AddrTxnAll:
		case dbtypes.AddrTxnCredit:
			addrData.Transactions = addrData.TxnsFunding
		case dbtypes.AddrTxnDebit:
			addrData.Transactions = addrData.TxnsSpending
		default:
			log.Warnf("Unknown address transaction type: %v", txnType)
		}

		// Transactions on current page
		addrData.NumTransactions = int64(len(addrData.Transactions))
		if addrData.NumTransactions > limitN {
			addrData.NumTransactions = limitN
		}

		// Query database for transaction details
		err = exp.explorerSource.FillAddressTransactions(addrData)
		if err != nil {
			log.Errorf("Unable to fill address %s transactions: %v", address, err)
			exp.ErrorPage(w, "Something went wrong...", "could not find transactions for that address", false)
			return
		}

		// Check for unconfirmed transactions
		addressOuts, numUnconfirmed, err := exp.blockData.UnconfirmedTxnsForAddress(address)
		if err != nil {
			log.Warnf("UnconfirmedTxnsForAddress failed for address %s: %v", address, err)
		}
		addrData.NumUnconfirmed = numUnconfirmed
		if addrData.UnconfirmedTxns == nil {
			addrData.UnconfirmedTxns = new(AddressTransactions)
		}
		uctxn := addrData.UnconfirmedTxns

		// Funding transactions (unconfirmed)
		for _, f := range addressOuts.Outpoints {
			fundingTx, ok := addressOuts.TxnsStore[f.Hash]
			if !ok {
				log.Errorf("An outpoint's transaction is not available in TxnStore.")
				continue
			}
			if fundingTx.Confirmed() {
				log.Errorf("An outpoint's transaction is unexpectedly confirmed.")
				continue
			}
			addrTx := &AddressTx{
				TxID:          fundingTx.Hash().String(),
				InOutID:       f.Index,
				FormattedSize: humanize.Bytes(uint64(fundingTx.Tx.SerializeSize())),
				Total:         txhelpers.TotalOutFromMsgTx(fundingTx.Tx).ToCoin(),
				ReceivedTotal: exccutil.Amount(fundingTx.Tx.TxOut[f.Index].Value).ToCoin(),
			}
			uctxn.Transactions = append(uctxn.Transactions, addrTx)
			uctxn.TxnsFunding = append(uctxn.TxnsFunding, addrTx)
		}

		// Spending transactions (unconfirmed)
		for _, f := range addressOuts.PrevOuts {
			spendingTx, ok := addressOuts.TxnsStore[f.TxSpending]
			if !ok {
				log.Errorf("An outpoint's transaction is not available in TxnStore.")
				continue
			}
			if spendingTx.Confirmed() {
				log.Errorf("An outpoint's transaction is unexpectedly confirmed.")
				continue
			}
			addrTx := &AddressTx{
				TxID:          spendingTx.Hash().String(),
				InOutID:       uint32(f.InputIndex),
				FormattedSize: humanize.Bytes(uint64(spendingTx.Tx.SerializeSize())),
				Total:         txhelpers.TotalOutFromMsgTx(spendingTx.Tx).ToCoin(),
				SentTotal:     exccutil.Amount(spendingTx.Tx.TxIn[f.InputIndex].ValueIn).ToCoin(),
			}
			uctxn.Transactions = append(uctxn.Transactions, addrTx)
			uctxn.TxnsSpending = append(uctxn.TxnsSpending, addrTx)
		}
	}

	// Set page parameters
	addrData.Path = r.URL.Path
	addrData.Limit, addrData.Offset = limitN, offsetAddrOuts
	addrData.TxnType = txnType.String()

	confirmHeights := make([]int64, len(addrData.Transactions))
	for i, v := range addrData.Transactions {
		confirmHeights[i] = exp.NewBlockData.Height - int64(v.Confirmations)
	}

	pageData := AddressPageData{
		Data:          addrData,
		ConfirmHeight: confirmHeights,
		Version:       exp.Version,
		NetName:       exp.NetName,
	}
	str, err := exp.templates.execTemplateToString("address", pageData)
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Turbolinks-Location", r.URL.RequestURI())
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// DecodeTxPage handles the "decode/broadcast transaction" page. The actual
// decoding or broadcasting is handled by the websocket hub.
func (exp *explorerUI) DecodeTxPage(w http.ResponseWriter, r *http.Request) {
	str, err := exp.templates.execTemplateToString("rawtx", struct {
		Version string
		NetName string
	}{
		exp.Version,
		exp.NetName,
	})
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing, that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// Search implements a primitive search algorithm by checking if the value in
// question is a block index, block hash, address hash or transaction hash and
// redirects to the appropriate page or displays an error
func (exp *explorerUI) Search(w http.ResponseWriter, r *http.Request) {
	searchStr := r.URL.Query().Get("search")
	if searchStr == "" {
		exp.ErrorPage(w, "search failed", "Nothing was searched for", true)
		return
	}

	// Attempt to get a block hash by calling GetBlockHash to see if the value
	// is a block index and then redirect to the block page if it is
	idx, err := strconv.ParseInt(searchStr, 10, 0)
	if err == nil {
		_, err = exp.blockData.GetBlockHash(idx)
		if err == nil {
			http.Redirect(w, r, "/block/"+searchStr, http.StatusPermanentRedirect)
			return
		}
		exp.ErrorPage(w, "search failed", "Block "+searchStr+" has not yet been mined", true)
		return
	}

	// Call GetExplorerAddress to see if the value is an address hash and
	// then redirect to the address page if it is
	address := exp.blockData.GetExplorerAddress(searchStr, 1, 0)
	if address != nil {
		http.Redirect(w, r, "/address/"+searchStr, http.StatusPermanentRedirect)
		return
	}

	// Check if the value is a valid hash
	if _, err = chainhash.NewHashFromStr(searchStr); err != nil {
		exp.ErrorPage(w, "search failed", "Couldn't find any address "+searchStr, true)
		return
	}

	// Attempt to get a block index by calling GetBlockHeight to see if the
	// value is a block hash and then redirect to the block page if it is
	_, err = exp.blockData.GetBlockHeight(searchStr)
	if err == nil {
		http.Redirect(w, r, "/block/"+searchStr, http.StatusPermanentRedirect)
		return
	}

	// Call GetExplorerTx to see if the value is a transaction hash and then
	// redirect to the tx page if it is
	tx := exp.blockData.GetExplorerTx(searchStr)
	if tx != nil {
		http.Redirect(w, r, "/tx/"+searchStr, http.StatusPermanentRedirect)
		return
	}
	exp.ErrorPage(w, "search failed", "Could not find any transaction or block "+searchStr, true)
}

// ErrorPage provides a way to show error on the pages without redirecting
func (exp *explorerUI) ErrorPage(w http.ResponseWriter, code string, message string, notFound bool) {
	str, err := exp.templates.execTemplateToString("error", struct {
		ErrorCode   string
		ErrorString string
		Version     string
		NetName     string
	}{
		code,
		message,
		exp.Version,
		exp.NetName,
	})
	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		str = "Something went very wrong if you can see this, try refreshing"
	}
	w.Header().Set("Content-Type", "text/html")
	if notFound {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	io.WriteString(w, str)
}

// NotFound wraps ErrorPage to display a 404 page
func (exp *explorerUI) NotFound(w http.ResponseWriter, r *http.Request) {
	exp.ErrorPage(w, "Not found", "Cannot find page: "+r.URL.Path, true)
}

// ParametersPage is the page handler for the "/parameters" path
func (exp *explorerUI) ParametersPage(w http.ResponseWriter, r *http.Request) {
	cp := exp.ChainParams
	addrPrefix := AddressPrefixes(cp)
	actualTicketPoolSize := int64(cp.TicketPoolSize * cp.TicketsPerBlock)
	ecp := ExtendedChainParams{
		Params:               cp,
		AddressPrefix:        addrPrefix,
		ActualTicketPoolSize: actualTicketPoolSize,
	}

	str, err := exp.templates.execTemplateToString("parameters", struct {
		Cp      ExtendedChainParams
		Version string
		NetName string
	}{
		ecp,
		exp.Version,
		exp.NetName,
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		exp.ErrorPage(w, "Something went wrong...", "and it's not your fault, try refreshing... that usually fixes things", false)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}
