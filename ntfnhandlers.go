// Copyright (c) 2018 The ExchangeCoin team
// Copyright (c) 2017, Jonathan Chappelow
// See LICENSE for details.

package main

import (
	"strings"
	"sync"
	"time"

	"github.com/EXCCoin/exccd/chaincfg/chainhash"
	"github.com/EXCCoin/exccd/exccutil"
	"github.com/EXCCoin/exccd/rpcclient"
	"github.com/EXCCoin/exccd/wire"
	"github.com/EXCCoin/exccdata/blockdata"
	"github.com/EXCCoin/exccdata/db/exccsqlite"
	"github.com/EXCCoin/exccdata/mempool"
	"github.com/EXCCoin/exccdata/stakedb"
	"github.com/EXCCoin/exccwallet/wallet/udb"
)

func registerNodeNtfnHandlers(exccdClient *rpcclient.Client) *ContextualError {
	var err error
	// Register for block connection and chain reorg notifications.
	if err = exccdClient.NotifyBlocks(); err != nil {
		return newContextualError("block notification "+
			"registration failed", err)
	}

	// Register for stake difficulty change notifications.
	// if err = exccdClient.NotifyStakeDifficulty(); err != nil {
	// 	return newContextualError("stake difficulty change "+
	// 		"notification registration failed", err)
	// }

	// Register for tx accepted into mempool ntfns
	if err = exccdClient.NotifyNewTransactions(false); err != nil {
		return newContextualError("new transaction "+
			"notification registration failed", err)
	}

	// For OnNewTickets
	//  Commented since there is a bug in rpcclient/notify.go
	// exccdClient.NotifyNewTickets()

	if err = exccdClient.NotifyWinningTickets(); err != nil {
		return newContextualError("winning ticket "+
			"notification registration failed", err)
	}

	// Register a Tx filter for addresses (receiving).  The filter applies to
	// OnRelevantTxAccepted.
	// TODO: register outpoints (third argument).
	// if len(addresses) > 0 {
	// 	if err = exccdClient.LoadTxFilter(true, addresses, nil); err != nil {
	// 		return newContextualError("load tx filter failed", err)
	// 	}
	// }

	return nil
}

type blockHashHeight struct {
	hash   chainhash.Hash
	height int64
}

type collectionQueue struct {
	sync.Mutex
	q            chan *blockHashHeight
	syncHandlers []func(hash *chainhash.Hash)
}

// NewCollectionQueue creates a new collectionQueue with a queue channel large
// enough for 10 million block pointers.
func NewCollectionQueue() *collectionQueue {
	return &collectionQueue{
		q: make(chan *blockHashHeight, 1e7),
	}
}

func (q *collectionQueue) SetSynchronousHandlers(syncHandlers []func(hash *chainhash.Hash)) {
	q.syncHandlers = syncHandlers
}

func (q *collectionQueue) ProcessBlocks() {
	// process queued blocks one at a time
	for bh := range q.q {
		hash := bh.hash
		height := bh.height

		start := time.Now()

		// Run synchronous block connected handlers in order
		for _, h := range q.syncHandlers {
			h(&hash)
		}

		log.Debugf("Synchronous handlers of collectionQueue.ProcessBlocks() completed in %v", time.Since(start))

		// Signal to mempool monitor that a block was mined
		select {
		case ntfnChans.newTxChan <- &mempool.NewTx{
			Hash: nil,
			T:    time.Now(),
		}:
		default:
		}

		// API status update handler
		select {
		case ntfnChans.updateStatusNodeHeight <- uint32(height):
		default:
		}
	}
}

// func (q *collectionQueue) PushBlock(b *blockHashHeight) {
// 	q.blockQueue = append(q.blockQueue, b)
// }

// func (q *collectionQueue) PopBlock() *blockHashHeight {
// 	if len(q.blockQueue) == 0 {
// 		return nil
// 	}
// 	b := q.blockQueue[0]
// 	q.blockQueue = q.blockQueue[1:]
// 	return b
// }

// Define notification handlers
func makeNodeNtfnHandlers(cfg *config) (*rpcclient.NotificationHandlers, *collectionQueue) {
	blockQueue := NewCollectionQueue()
	go blockQueue.ProcessBlocks()
	return &rpcclient.NotificationHandlers{
		OnBlockConnected: func(blockHeaderSerialized []byte, transactions [][]byte) {
			blockHeader := new(wire.BlockHeader)
			err := blockHeader.FromBytes(blockHeaderSerialized)
			if err != nil {
				log.Error("Failed to serialize blockHeader in new block notification.")
			}
			height := int32(blockHeader.Height)
			hash := blockHeader.BlockHash()

			// queue this block
			blockQueue.q <- &blockHashHeight{
				hash:   hash,
				height: int64(height),
			}
		},
		OnReorganization: func(oldHash *chainhash.Hash, oldHeight int32,
			newHash *chainhash.Hash, newHeight int32) {
			// Send reorg data to exccsqlite's monitor
			select {
			case ntfnChans.reorgChanWiredDB <- &exccsqlite.ReorgData{
				OldChainHead:   *oldHash,
				OldChainHeight: oldHeight,
				NewChainHead:   *newHash,
				NewChainHeight: newHeight,
			}:
			default:
			}
			// Send reorg data to blockdata's monitor (so that it stops collecting)
			select {
			case ntfnChans.reorgChanBlockData <- &blockdata.ReorgData{
				OldChainHead:   *oldHash,
				OldChainHeight: oldHeight,
				NewChainHead:   *newHash,
				NewChainHeight: newHeight,
			}:
			default:
			}
			// Send reorg data to stakedb's monitor
			select {
			case ntfnChans.reorgChanStakeDB <- &stakedb.ReorgData{
				OldChainHead:   *oldHash,
				OldChainHeight: oldHeight,
				NewChainHead:   *newHash,
				NewChainHeight: newHeight,
			}:
			default:
			}
		},
		// Not too useful since this notifies on every block
		// OnStakeDifficulty: func(hash *chainhash.Hash, height int64,
		// 	stakeDiff int64) {
		// 	select {
		// 	case ntfnChans.stakeDiffChan <- stakeDiff:
		// 	default:
		// 	}
		// },
		// TODO
		OnWinningTickets: func(blockHash *chainhash.Hash, blockHeight int64,
			tickets []*chainhash.Hash) {
			var txstr []string
			for _, t := range tickets {
				txstr = append(txstr, t.String())
			}
			log.Debugf("Winning tickets: %v", strings.Join(txstr, ", "))
		},
		// maturing tickets. Thanks for fixing the tickets type bug, jolan!
		OnNewTickets: func(hash *chainhash.Hash, height int64, stakeDiff int64,
			tickets []*chainhash.Hash) {
			for _, tick := range tickets {
				log.Tracef("Mined new ticket: %v", tick.String())
			}
		},
		// OnRelevantTxAccepted is invoked when a transaction containing a
		// registered address is inserted into mempool.
		OnRelevantTxAccepted: func(transaction []byte) {
			rec, err := udb.NewTxRecord(transaction, time.Now())
			if err != nil {
				return
			}
			tx := exccutil.NewTx(&rec.MsgTx)
			txHash := rec.Hash
			select {
			case ntfnChans.relevantTxMempoolChan <- tx:
				log.Debugf("Detected transaction %v in mempool containing registered address.",
					txHash.String())
			default:
			}
		},
		// OnTxAccepted is invoked when a transaction is accepted into the
		// memory pool.  It will only be invoked if a preceding call to
		// NotifyNewTransactions with the verbose flag set to false has been
		// made to register for the notification and the function is non-nil.
		OnTxAccepted: func(hash *chainhash.Hash, amount exccutil.Amount) {
			// Just send the tx hash and let the goroutine handle everything.
			select {
			case ntfnChans.newTxChan <- &mempool.NewTx{
				Hash: hash,
				T:    time.Now(),
			}:
			default:
				log.Warn("newTxChan buffer full!")
			}
			//log.Trace("Transaction accepted to mempool: ", hash, amount)
		},
		// Note: exccjson.TxRawResult is from getrawtransaction
		//OnTxAcceptedVerbose: func(txDetails *exccjson.TxRawResult) {
		//txDetails.Hex
		//log.Info("Transaction accepted to mempool: ", txDetails.Txid)
		//},
	}, blockQueue
}
