// Copyright (c) 2018, The Decred developers
// Copyright (c) 2018, The dcrdata developers
// Copyright (c) 2017, Jonathan Chappelow
// See LICENSE for details.

package stakedb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/EXCCoin/exccd/blockchain/stake"
	"github.com/EXCCoin/exccd/chaincfg"
	"github.com/EXCCoin/exccd/chaincfg/chainhash"
	"github.com/EXCCoin/exccd/database"
	"github.com/EXCCoin/exccd/exccutil"
	"github.com/EXCCoin/exccd/rpcclient"
	"github.com/EXCCoin/exccd/wire"
	apitypes "github.com/EXCCoin/exccdata/api/types"
	"github.com/EXCCoin/exccdata/rpcutils"
	"github.com/EXCCoin/exccdata/txhelpers"
	"github.com/oleiade/lane"
)

// PoolInfoCache contains a map of block hashes to ticket pool info data at that
// block height.
type PoolInfoCache struct {
	sync.RWMutex
	poolInfo    map[chainhash.Hash]*apitypes.TicketPoolInfo
	expireQueue *lane.Queue
	maxSize     int
}

// NewPoolInfoCache constructs a new PoolInfoCache, and is needed to initialize
// the internal map.
func NewPoolInfoCache(size int) *PoolInfoCache {
	return &PoolInfoCache{
		poolInfo:    make(map[chainhash.Hash]*apitypes.TicketPoolInfo),
		expireQueue: lane.NewQueue(),
		maxSize:     size,
	}
}

// Get attempts to fetch the ticket pool info for a given block hash, returning
// a *apitypes.TicketPoolInfo, and a bool indicating if the hash was found in
// the map.
func (c *PoolInfoCache) Get(hash chainhash.Hash) (*apitypes.TicketPoolInfo, bool) {
	c.RLock()
	defer c.RUnlock()
	tpi, ok := c.poolInfo[hash]
	return tpi, ok
}

// Set stores the ticket pool info for the given hash in the pool info cache.
func (c *PoolInfoCache) Set(hash chainhash.Hash, p *apitypes.TicketPoolInfo) {
	c.Lock()
	defer c.Unlock()
	c.poolInfo[hash] = p
	c.expireQueue.Enqueue(hash)
	if c.expireQueue.Size() >= c.maxSize {
		expireHash := c.expireQueue.Dequeue().(chainhash.Hash)
		delete(c.poolInfo, expireHash)
	}
}

// SetCapacity sets the cache capacity to the specified number of elements. If
// the new capacity is smaller than the current cache size, elements are
// automatically evicted until the desired size is reached.
func (c *PoolInfoCache) SetCapacity(cap int) error {
	c.Lock()
	defer c.Unlock()
	c.maxSize = cap
	for c.expireQueue.Size() >= c.maxSize {
		expireHash, ok := c.expireQueue.Dequeue().(chainhash.Hash)
		if !ok {
			return fmt.Errorf("failed to reduce pool cache capacity")
		}
		delete(c.poolInfo, expireHash)
	}
	return nil
}

// StakeDatabase models data for the stake database
type StakeDatabase struct {
	params          *chaincfg.Params
	NodeClient      *rpcclient.Client
	nodeMtx         sync.RWMutex
	StakeDB         database.DB
	BestNode        *stake.Node
	blkMtx          sync.RWMutex
	blockCache      map[int64]*exccutil.Block
	liveTicketMtx   sync.RWMutex
	liveTicketCache map[chainhash.Hash]int64
	poolValue       int64
	poolInfo        *PoolInfoCache
	PoolDB          *TicketPool

	// clients may register for notification when new blocks are connected via
	// WaitForHeight. The clients' channels are stored in heightWaiters.
	waitMtx       sync.Mutex
	heightWaiters map[int64][]chan *chainhash.Hash
}

const (
	// dbType is the database backend type to use
	dbType = "ffldb"
	// DefaultStakeDbName is the default name of the stakedb database folder
	DefaultStakeDbName = "stakenodes"
	// DefaultTicketPoolDbName is the default name of the ticket pool database
	DefaultTicketPoolDbName = "ticket_pool.db"
)

// NewStakeDatabase creates a StakeDatabase instance, opening or creating a new
// ffldb-backed stake database, and loads all live tickets into a cache.
func NewStakeDatabase(client *rpcclient.Client, params *chaincfg.Params,
	dbFolder string) (*StakeDatabase, error) {
	// Create DB folder
	err := os.MkdirAll(dbFolder, 0700)
	if err != nil {
		return nil, fmt.Errorf("unable to create DB folder: %v", err)
	}
	log.Infof("Loading ticket pool DB. This may take a minute...")
	ticketPoolDBPath := filepath.Join(dbFolder, DefaultTicketPoolDbName)
	poolDB, err := NewTicketPool(ticketPoolDBPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open ticket pool DB: %v", err)
	}
	sDB := &StakeDatabase{
		params:          params,
		NodeClient:      client,
		blockCache:      make(map[int64]*exccutil.Block),
		liveTicketCache: make(map[chainhash.Hash]int64, params.TicketPoolSize*(params.TicketsPerBlock+1)),
		poolInfo:        NewPoolInfoCache(513),
		PoolDB:          poolDB,
		heightWaiters:   make(map[int64][]chan *chainhash.Hash),
	}

	// Put the genesis block in the pool info cache since stakedb starts with
	// genesis. Hence it will never be connected, how TPI is usually cached.
	sDB.poolInfo.Set(*params.GenesisHash, &apitypes.TicketPoolInfo{})

	stakeDBPath := filepath.Join(dbFolder, DefaultStakeDbName)
	if err = sDB.Open(stakeDBPath); err != nil {
		return nil, err
	}

	// Check if stake DB and ticket pool DB are at the same height, and attempt
	// to recover.
	heightStakeDB, heightTicketPool := int64(sDB.Height()), sDB.PoolDB.Tip()
	if heightStakeDB != heightTicketPool {
		// Roll stake DB back to the height of the ticket pool DB
		for heightStakeDB > heightTicketPool {
			if err = sDB.DisconnectBlock(); err != nil {
				return nil, fmt.Errorf("failed to disconnect block: %v", err)
			}
			heightStakeDB, heightTicketPool = int64(sDB.Height()), sDB.PoolDB.Tip()
		}

		// Trim ticket pool DB back to the height of the stake DB
		for heightTicketPool > heightStakeDB {
			heightTicketPool, _ = sDB.PoolDB.Trim()
		}
		if heightTicketPool != heightStakeDB {
			return nil, fmt.Errorf("unable to return stake DB height (%d) to ticket pool height (%d)",
				heightStakeDB, heightTicketPool)
		}
	}

	log.Infof("Advancing ticket pool DB to tip via diffs...")
	if err = poolDB.AdvanceToTip(); err != nil {
		return nil, fmt.Errorf("failed to advance ticket pool DB to tip: %v", err)
	}

	// Live tickets from exccdata's stake Node's perspective
	liveTickets := sDB.BestNode.LiveTickets()

	log.Info("Pre-populating live ticket cache and computing pool value...")

	// Send all the live ticket requests
	type promiseGetRawTransaction struct {
		result rpcclient.FutureGetRawTransactionResult
		ticket chainhash.Hash
	}
	promisesGetRawTransaction := make([]promiseGetRawTransaction, 0, len(liveTickets))

	// Send all the live ticket requests
	for _, hash := range liveTickets {
		promisesGetRawTransaction = append(promisesGetRawTransaction, promiseGetRawTransaction{
			result: sDB.NodeClient.GetRawTransactionAsync(&hash),
			ticket: hash,
		})
	}

	// Receive the live ticket tx results
	for _, p := range promisesGetRawTransaction {
		ticketTx, err := p.result.Receive()
		if err != nil {
			log.Error(err)
			continue
		}
		if !ticketTx.Hash().IsEqual(&p.ticket) {
			panic(fmt.Sprintf("Failed to receive Tx details for requested ticket hash: %v, %v", p.ticket, ticketTx.Hash()))
		}

		value := ticketTx.MsgTx().TxOut[0].Value
		sDB.poolValue += value
		sDB.liveTicketCache[p.ticket] = value
	}

	return sDB, nil
}

// LockStakeNode locks the StakeNode from functions that respect the mutex.
func (db *StakeDatabase) LockStakeNode() {
	db.nodeMtx.RLock()
}

// UnlockStakeNode unlocks the StakeNode for functions that respect the mutex.
func (db *StakeDatabase) UnlockStakeNode() {
	db.nodeMtx.RUnlock()
}

// WaitForHeight provides a notification channel to which the hash of the block
// at the requested height will be sent when it becomes available.
func (db *StakeDatabase) WaitForHeight(height int64) chan *chainhash.Hash {
	dbHeight := int64(db.Height())
	waitChan := make(chan *chainhash.Hash, 1)
	if dbHeight > height {
		defer func() { waitChan <- nil }()
		return waitChan
	} else if dbHeight == height {
		block, _ := db.block(height)
		if block == nil {
			panic("broken StakeDatabase")
		}
		defer func() { go db.signalWaiters(height, block.Hash()) }()
	}
	db.waitMtx.Lock()
	db.heightWaiters[height] = append(db.heightWaiters[height], waitChan)
	db.waitMtx.Unlock()
	return waitChan
}

func (db *StakeDatabase) signalWaiters(height int64, blockhash *chainhash.Hash) {
	db.waitMtx.Lock()
	defer db.waitMtx.Unlock()
	waitChans := db.heightWaiters[height]
	for _, c := range waitChans {
		select {
		case c <- blockhash:
		default:
			panic(fmt.Sprintf("unable to signal block with hash %v at height %d", blockhash, height))
		}
	}

	delete(db.heightWaiters, height)
}

// Height gets the block height of the best stake node.  It is thread-safe,
// unlike using db.BestNode.Height(), and checks that the stake database is
// opened first.
func (db *StakeDatabase) Height() uint32 {
	db.nodeMtx.RLock()
	defer db.nodeMtx.RUnlock()
	if db == nil || db.BestNode == nil {
		log.Error("Stake database not yet opened")
		return 0
	}
	return db.BestNode.Height()
}

// block first tries to find the block at the input height in cache, and if that
// fails it will request it from the node RPC client. Don't use this casually
// since reorganization may redefine a block at a given height.
func (db *StakeDatabase) block(ind int64) (*exccutil.Block, bool) {
	db.blkMtx.RLock()
	block, ok := db.blockCache[ind]
	db.blkMtx.RUnlock()
	//log.Info(ind, block, ok)
	if !ok {
		var err error
		block, _, err = rpcutils.GetBlock(ind, db.NodeClient)
		if err != nil {
			log.Error(err)
			return nil, false
		}
	}
	return block, ok
}

// ForgetBlock deletes the block with the input height from the block cache.
func (db *StakeDatabase) ForgetBlock(ind int64) {
	db.blkMtx.Lock()
	defer db.blkMtx.Unlock()
	delete(db.blockCache, ind)
}

// ConnectBlockHash is a wrapper for ConnectBlock. For the input block hash, it
// gets the block from the node RPC client and calls ConnectBlock.
func (db *StakeDatabase) ConnectBlockHash(hash *chainhash.Hash) (*exccutil.Block, error) {
	msgBlock, err := db.NodeClient.GetBlock(hash)
	if err != nil {
		return nil, err
	}
	block := exccutil.NewBlock(msgBlock)
	return block, db.ConnectBlock(block)
}

// ConnectBlock connects the input block to the tip of the stake DB and updates
// the best stake node. This exported function gets any revoked and spend
// tickets from the input block, and any maturing tickets from the past block in
// which those tickets would be found, and passes them to connectBlock.
func (db *StakeDatabase) ConnectBlock(block *exccutil.Block) error {
	height := block.Height()
	maturingHeight := height - int64(db.params.TicketMaturity)

	var maturingTickets []chainhash.Hash
	if maturingHeight >= 0 {
		maturingBlock, wasCached := db.block(maturingHeight)
		if wasCached {
			db.ForgetBlock(maturingHeight)
		}
		maturingTickets, _ = txhelpers.TicketsInBlock(maturingBlock)
	}

	db.blkMtx.Lock()
	db.blockCache[height] = block
	db.blkMtx.Unlock()

	revokedTickets := txhelpers.RevokedTicketsInBlock(block)
	votedTickets := txhelpers.TicketsSpentInBlock(block)

	db.nodeMtx.Lock()
	defer db.nodeMtx.Unlock()
	bestNodeHeight := int64(db.BestNode.Height())
	if height <= bestNodeHeight {
		return fmt.Errorf("cannot connect block height %d at height %d", height, bestNodeHeight)
	}

	// Who is supposed to vote in the block we are about to connect
	winners := db.BestNode.Winners()

	// connect it, updating BestNode
	err := db.connectBlock(block, votedTickets, revokedTickets, maturingTickets)
	if err != nil {
		return err
	}

	// Expiring tickets
	expiring := db.BestNode.ExpiredByBlock()

	// Tickets leaving the live ticket pool = winners + expires
	liveOut := append(winners, expiring...)
	// Tickets entering the pool = maturing tickets
	poolDiff := &PoolDiff{
		In:  maturingTickets,
		Out: liveOut,
	}

	defer func() { go db.signalWaiters(height, block.Hash()) }()

	// update liveTicketCache and poolValue
	db.applyDiff(*poolDiff)

	// Some sanity checks
	// db.liveTicketMtx.RLock()
	// liveTickets := make([]chainhash.Hash, 0, len(db.liveTicketCache))
	// for h := range db.liveTicketCache {
	// 	liveTickets = append(liveTickets, h)
	// }
	// db.liveTicketMtx.RUnlock()

	// Get ticket pool info at current best (just connected in stakedb) block,
	// and store it in the StakeDatabase's PoolInfoCache.
	// liveTicketsX := db.BestNode.LiveTickets()
	// if len(db.liveTicketCache) != len(liveTicketsX) {
	// 	log.Errorf("%d != %d", len(db.liveTicketCache), len(liveTicketsX))
	// }

	// Store TicketPoolInfo in the PoolInfoCache
	poolSize := int64(db.BestNode.PoolSize())
	winningTickets := db.BestNode.Winners()
	pib := db.makePoolInfo(db.poolValue, poolSize, winningTickets, uint32(height))
	db.poolInfo.Set(*block.Hash(), pib)

	// Append this ticket pool diff
	return db.PoolDB.Append(poolDiff, bestNodeHeight+1)
}

func (db *StakeDatabase) connectBlock(block *exccutil.Block, spent []chainhash.Hash,
	revoked []chainhash.Hash, maturing []chainhash.Hash) error {

	hB, err := block.BlockHeaderBytes()
	if err != nil {
		return fmt.Errorf("unable to serialize block header: %v", err)
	}

	bestNode, err := db.BestNode.ConnectNode(stake.CalcHash256PRNGIV(hB),
		spent, revoked, maturing)
	if err != nil {
		return err
	}
	if bestNode == nil {
		return fmt.Errorf("failed to ConnectNode at BestNode")
	}
	db.BestNode = bestNode

	return db.StakeDB.Update(func(dbTx database.Tx) error {
		return stake.WriteConnectedBestNode(dbTx, db.BestNode, *block.Hash())
	})
}

// applyDiff updates liveTicketCache and poolValue for the given PoolDiff.
func (db *StakeDatabase) applyDiff(poolDiff PoolDiff) {
	db.liveTicketMtx.Lock()
	for _, hash := range poolDiff.In {
		_, ok := db.liveTicketCache[hash]
		if ok {
			log.Error("Just tried to add a ticket (%v) to the pool, but it was already there!", hash)
			continue
		}

		tx, err := db.NodeClient.GetRawTransaction(&hash)
		if err != nil {
			log.Errorf("Unable to get transaction %v: %v\n", hash, err)
			continue
		}
		// This isn't quite right for pool tickets where the small
		// pool fees are included in vout[0], but it's close.
		val := tx.MsgTx().TxOut[0].Value
		db.liveTicketCache[hash] = val
		db.poolValue += val
	}

	for _, h := range poolDiff.Out {
		valOut, ok := db.liveTicketCache[h]
		if !ok {
			log.Errorf("Didn't find %v in live ticket store!", h)
			continue
		}
		db.poolValue -= valOut
		delete(db.liveTicketCache, h)
	}
	db.liveTicketMtx.Unlock()
}

// undoDiff is like applyDiff except it swaps In and Out in specifed PoolDiff.
func (db *StakeDatabase) undoDiff(poolDiff PoolDiff) {
	db.applyDiff(PoolDiff{
		In:  poolDiff.Out,
		Out: poolDiff.In,
	})
}

// SetPoolInfo stores the ticket pool info for the given hash in the pool info
// cache.
func (db *StakeDatabase) SetPoolInfo(blockHash chainhash.Hash, tpi *apitypes.TicketPoolInfo) {
	db.poolInfo.Set(blockHash, tpi)
}

// SetPoolCacheCapacity sets the pool info cache capacity to the specified
// number of elements.
func (db *StakeDatabase) SetPoolCacheCapacity(cap int) error {
	return db.poolInfo.SetCapacity(cap)
}

// DisconnectBlock attempts to disconnect the current best block from the stake
// DB and updates the best stake node.
func (db *StakeDatabase) DisconnectBlock() error {
	db.nodeMtx.Lock()
	defer db.nodeMtx.Unlock()

	return db.disconnectBlock()
}

// disconnectBlock is the non-thread-safe version of DisconnectBlock.
func (db *StakeDatabase) disconnectBlock() error {
	childHeight := db.BestNode.Height()
	parentBlock, err := db.dbPrevBlock()
	if err != nil {
		return err
	}
	if parentBlock.Height() != int64(childHeight)-1 {
		panic("BestNode and stake DB are inconsistent")
	}

	// Trim the ticket pool db to the same height as stake.Node
	var undoDiffs []PoolDiff
	poolDBTip := db.PoolDB.tip
	for poolDBTip > parentBlock.Height() {
		newTip, undoDiff := db.PoolDB.Trim()
		undoDiffs = append(undoDiffs, undoDiff)
		if newTip >= poolDBTip {
			panic("unable to trim pool DB!")
		}
		poolDBTip = newTip
	}

	// Update liveTicketCache and poolValue
	for i := range undoDiffs {
		db.undoDiff(undoDiffs[i])
	}

	log.Debugf("Disconnecting block %d.", childHeight)
	childUndoData := append(stake.UndoTicketDataSlice(nil), db.BestNode.UndoData()...)

	// previous best node
	hB, errx := parentBlock.BlockHeaderBytes()
	if errx != nil {
		return fmt.Errorf("unable to serialize block header: %v", errx)
	}
	parentIV := stake.CalcHash256PRNGIV(hB)

	var parentStakeNode *stake.Node
	err = db.StakeDB.View(func(dbTx database.Tx) error {
		var errLocal error
		parentStakeNode, errLocal = db.BestNode.DisconnectNode(parentIV, nil, nil, dbTx)
		return errLocal
	})
	if err != nil {
		return err
	}
	if parentStakeNode == nil {
		return fmt.Errorf("failed to DisconnectNode at BestNode")
	}
	db.BestNode = parentStakeNode

	return db.StakeDB.Update(func(dbTx database.Tx) error {
		return stake.WriteDisconnectedBestNode(dbTx, parentStakeNode,
			*parentBlock.Hash(), childUndoData)
	})
}

// DisconnectBlocks disconnects N blocks from the head of the chain.
func (db *StakeDatabase) DisconnectBlocks(count int64) error {
	db.nodeMtx.Lock()
	defer db.nodeMtx.Unlock()

	for i := int64(0); i < count; i++ {
		if err := db.disconnectBlock(); err != nil {
			return err
		}
	}

	return nil
}

// Open attempts to open an existing stake database, and will create a new one
// if one does not exist.
func (db *StakeDatabase) Open(dbName string) error {
	db.nodeMtx.Lock()
	defer db.nodeMtx.Unlock()

	// Create a new database to store the accepted stake node data into.
	var isFreshDB bool
	var err error
	db.StakeDB, err = database.Open(dbType, dbName, db.params.Net)
	if err != nil {
		if strings.Contains(err.Error(), "resource temporarily unavailable") ||
			strings.Contains(err.Error(), "is being used by another process") {
			return fmt.Errorf("Stake DB already opened. exccdata running?")
		}
		if strings.Contains(err.Error(), "does not exist") {
			log.Info("Creating new stake DB.")
		} else {
			log.Infof("Unable to open stake DB (%v). Removing and creating new.", err)
			_ = os.RemoveAll(dbName)
		}

		db.StakeDB, err = database.Create(dbType, dbName, db.params.Net)
		if err != nil {
			// do not return nil interface, but interface of nil DB
			return fmt.Errorf("error creating database.DB: %v", err)
		}
		isFreshDB = true
	}

	// Load the best block from stake db
	err = db.StakeDB.View(func(dbTx database.Tx) error {
		v := dbTx.Metadata().Get([]byte("stakechainstate"))
		if v == nil {
			return fmt.Errorf("missing key for chain state data")
		}

		var stakeDBHash chainhash.Hash
		copy(stakeDBHash[:], v[:chainhash.HashSize])
		offset := chainhash.HashSize
		stakeDBHeight := binary.LittleEndian.Uint32(v[offset : offset+4])

		var errLocal error
		msgBlock, errLocal := db.NodeClient.GetBlock(&stakeDBHash)
		if errLocal != nil {
			return fmt.Errorf("GetBlock failed (%s): %v", stakeDBHash, errLocal)
		}
		header := msgBlock.Header

		db.BestNode, errLocal = stake.LoadBestNode(dbTx, stakeDBHeight,
			stakeDBHash, header, db.params)
		return errLocal
	})
	if err != nil {
		if !isFreshDB {
			log.Errorf("Error reading from database (%v).  Reinitializing.", err)
		}
		err = db.StakeDB.Update(func(dbTx database.Tx) error {
			var errLocal error
			db.BestNode, errLocal = stake.InitDatabaseState(dbTx, db.params)
			return errLocal
		})
		log.Debug("Initialized new stake db.")
	} else {
		log.Debug("Opened existing stake db.")
	}

	return err
}

// Close will close the ticket pool and stake databases.
func (db *StakeDatabase) Close() error {
	db.nodeMtx.Lock()
	defer db.nodeMtx.Unlock()
	err1 := db.PoolDB.Close()
	err2 := db.StakeDB.Close()
	if err1 == nil {
		return err2
	}
	if err2 == nil {
		return err1
	}
	return fmt.Errorf("%v + %v", err1, err2)
}

func (db *StakeDatabase) expires() ([]chainhash.Hash, []bool) {
	// revoked includes expired ticket and missed votes that were revoked
	revoked := db.BestNode.RevokedTickets()
	// unrevoked includes expired and missed that have not been revoked
	unrevoked := db.BestNode.MissedTickets()

	var expires []chainhash.Hash
	var spent []bool
	for _, tkt := range revoked {
		if db.BestNode.ExistsExpiredTicket(*tkt) {
			expires = append(expires, *tkt)
			spent = append(spent, true)
		}
	}
	for _, tkt := range unrevoked {
		if db.BestNode.ExistsExpiredTicket(tkt) {
			expires = append(expires, tkt)
			spent = append(spent, false)
		}
	}
	return expires, spent
}

// PoolInfoBest computes ticket pool value using the database and, if needed, the
// node RPC client to fetch ticket values that are not cached. Returned are a
// structure including ticket pool value, size, and average value.
func (db *StakeDatabase) PoolInfoBest() *apitypes.TicketPoolInfo {
	db.nodeMtx.RLock()
	if db.BestNode == nil {
		db.nodeMtx.RUnlock()
		log.Errorf("PoolInfoBest: BestNode is nil!")
		return nil
	}
	poolSize := db.BestNode.PoolSize()
	//liveTickets := db.BestNode.LiveTickets()
	winningTickets := db.BestNode.Winners()
	height := db.BestNode.Height()
	// expiredTickets, expireRevoked := db.expires()
	db.nodeMtx.RUnlock()

	return db.makePoolInfo(db.poolValue, int64(poolSize), winningTickets, height) // db.calcPoolInfo(liveTickets, winningTickets, height)
}

func (db *StakeDatabase) makePoolInfo(poolValue, poolSize int64,
	winningTickets []chainhash.Hash, height uint32) *apitypes.TicketPoolInfo {
	poolCoin := exccutil.Amount(poolValue).ToCoin()
	valAvg := 0.0
	if poolSize > 0 {
		valAvg = poolCoin / float64(poolSize)
	}

	winners := make([]string, 0, len(winningTickets))
	for _, winner := range winningTickets {
		winners = append(winners, winner.String())
	}

	return &apitypes.TicketPoolInfo{
		Height:  height,
		Size:    uint32(poolSize),
		Value:   poolCoin,
		ValAvg:  valAvg,
		Winners: winners,
	}
}

func (db *StakeDatabase) calcPoolInfo(liveTickets, winningTickets []chainhash.Hash, height uint32) *apitypes.TicketPoolInfo {
	poolSize := len(liveTickets)
	db.liveTicketMtx.Lock()
	var poolValue int64
	for _, hash := range liveTickets {
		val, ok := db.liveTicketCache[hash]
		if !ok {
			tx, err := db.NodeClient.GetRawTransaction(&hash)
			if err != nil {
				log.Errorf("Unable to get transaction %v: %v\n", hash, err)
				continue
			}
			// This isn't quite right for pool tickets where the small
			// pool fees are included in vout[0], but it's close.
			val = tx.MsgTx().TxOut[0].Value
			db.liveTicketCache[hash] = val
		}
		poolValue += val
	}
	db.liveTicketMtx.Unlock()

	poolCoin := exccutil.Amount(poolValue).ToCoin()
	valAvg := 0.0
	if len(liveTickets) > 0 {
		valAvg = poolCoin / float64(poolSize)
	}

	winners := make([]string, 0, len(winningTickets))
	for _, winner := range winningTickets {
		winners = append(winners, winner.String())
	}

	return &apitypes.TicketPoolInfo{
		Height:  height,
		Size:    uint32(poolSize),
		Value:   poolCoin,
		ValAvg:  valAvg,
		Winners: winners,
	}
}

// PoolInfo attempts to fetch the ticket pool info for the specified block hash
// from an internal pool info cache. If it is not found, you should attempt to
// use PoolInfoBest if the target block is at the tip of the chain.
func (db *StakeDatabase) PoolInfo(hash chainhash.Hash) (*apitypes.TicketPoolInfo, bool) {
	return db.poolInfo.Get(hash)
}

// PoolSize returns the ticket pool size in the best node of the stake database
func (db *StakeDatabase) PoolSize() int {
	db.nodeMtx.Lock()
	defer db.nodeMtx.Unlock()
	return db.BestNode.PoolSize()
}

// PoolAtHeight gets the entire list of live tickets at the given chain height.
func (db *StakeDatabase) PoolAtHeight(height int64) ([]chainhash.Hash, error) {
	return db.PoolDB.Pool(height)
}

// PoolAtHash gets the entire list of live tickets at the given block hash.
func (db *StakeDatabase) PoolAtHash(hash chainhash.Hash) ([]chainhash.Hash, error) {
	header, err := db.NodeClient.GetBlockHeader(&hash)
	if err != nil {
		return nil, fmt.Errorf("GetBlockHeader failed: %v", err)
	}
	return db.PoolDB.Pool(int64(header.Height))
}

// DBState queries the stake database for the best block height and hash.
func (db *StakeDatabase) DBState() (uint32, *chainhash.Hash, error) {
	db.nodeMtx.RLock()
	defer db.nodeMtx.RUnlock()

	return db.dbState()
}

func (db *StakeDatabase) dbState() (uint32, *chainhash.Hash, error) {
	var stakeDBHeight uint32
	var stakeDBHash chainhash.Hash
	err := db.StakeDB.View(func(dbTx database.Tx) error {
		v := dbTx.Metadata().Get([]byte("stakechainstate"))
		if v == nil {
			return fmt.Errorf("missing key for chain state data")
		}

		copy(stakeDBHash[:], v[:chainhash.HashSize])
		offset := chainhash.HashSize
		stakeDBHeight = binary.LittleEndian.Uint32(v[offset : offset+4])

		return nil
	})
	return stakeDBHeight, &stakeDBHash, err
}

// DBTipBlockHeader gets the block header for the current best block in the
// stake database. It used DBState to get the best block hash, and the node RPC
// client to get the header.
func (db *StakeDatabase) DBTipBlockHeader() (*wire.BlockHeader, error) {
	_, hash, err := db.DBState()
	if err != nil {
		return nil, err
	}

	return db.NodeClient.GetBlockHeader(hash)
}

// DBPrevBlockHeader gets the block header for the previous best block in the
// stake database. It used DBState to get the best block hash, and the node RPC
// client to get the header.
func (db *StakeDatabase) DBPrevBlockHeader() (*wire.BlockHeader, error) {
	_, hash, err := db.DBState()
	if err != nil {
		return nil, err
	}

	parentHeader, err := db.NodeClient.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	return db.NodeClient.GetBlockHeader(&parentHeader.PrevBlock)
}

// DBTipBlock gets the exccutil.Block for the current best block in the stake
// database. It used DBState to get the best block hash, and the node RPC client
// to get the block itself.
func (db *StakeDatabase) DBTipBlock() (*exccutil.Block, error) {
	_, hash, err := db.DBState()
	if err != nil {
		return nil, err
	}

	return db.getBlock(hash)
}

// DBPrevBlock gets the exccutil.Block for the previous best block in the stake
// database. It used DBState to get the best block hash, and the node RPC client
// to get the block itself.
func (db *StakeDatabase) DBPrevBlock() (*exccutil.Block, error) {
	_, hash, err := db.DBState()
	if err != nil {
		return nil, err
	}

	parentHeader, err := db.NodeClient.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	return db.getBlock(&parentHeader.PrevBlock)
}

// dbPrevBlock is the non-thread-safe version of DBPrevBlock.
func (db *StakeDatabase) dbPrevBlock() (*exccutil.Block, error) {
	_, hash, err := db.dbState()
	if err != nil {
		return nil, err
	}

	parentHeader, err := db.NodeClient.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	return db.getBlock(&parentHeader.PrevBlock)
}

func (db *StakeDatabase) getBlock(hash *chainhash.Hash) (*exccutil.Block, error) {
	msgBlock, err := db.NodeClient.GetBlock(hash)
	if err == nil {
		return exccutil.NewBlock(msgBlock), nil
	}
	return nil, err
}
