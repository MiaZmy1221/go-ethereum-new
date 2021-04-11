package realtime

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	// "os"
	"math/big"
	// "github.com/ethereum/go-ethereum/trace"
	"github.com/ethereum/go-ethereum/eth/downloader"

	"errors"
	"sync"
	"sync/atomic"
	// "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

// Backend wraps all methods required for mining.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
}

// Miner creates blocks and searches for proof-of-work values.
// Simulates Miner, Simulator just executes the realtime transactions.
type Simulator struct {
	startCh  chan struct{}
	newTxsCh chan []*types.Transaction
	stopCh   chan struct{}

	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// to store all the transactions
	simTxPool   *SimTxPool	

	running		int32

	// to pretect the execution????	
	exe          sync.RWMutex

}

// New the simulator, do not worry 
func New(eth Backend, chainConfig *params.ChainConfig, engine consensus.Engine) *Simulator {
	simulator := &Simulator{
		chainConfig:        chainConfig,
		engine:             engine,
		eth:                eth,
		chain:              eth.BlockChain(),	
		simTxPool:   		NewSimTxPool(chainConfig, eth.BlockChain()),
		// running:			uint64(0),	
		// startCh: 			make(chan struct{}),
		startCh:            make(chan struct{}, 1),
		stopCh:  			make(chan struct{}),
		newTxsCh:			make(chan []*types.Transaction),
	}
	go simulator.loop()

	simulator.startCh <- struct{}{}

	return simulator
}


// isRunning returns an indicator whether worker is running or not.
func (simulator *Simulator) isRunning() bool {
	return atomic.LoadInt32(&simulator.running) == 1
}


func (simulator *Simulator) loop() {
	for {
		select {
		case <-simulator.startCh:
			atomic.StoreInt32(&simulator.running, 1)
		case <-simulator.stopCh:
			atomic.StoreInt32(&simulator.running, 0)
		case newTxs := <-simulator.newTxsCh:
			// for i, tx := range newTxs {
			// 	receipt_logs, newerr := simulator.ExecuteTransaction(tx)
			// }

			// How to deal with this area??????????????????????????????????????
			// like dealing with the first 100 transactions,????? by order???? 
			// right now, just execute the new added txs
			if simulator.isRunning() == true {
				simulator.exe.RLock()
				for _, tx := range newTxs {
					simulator.ExecuteTransaction(tx)
				}
				simulator.exe.RUnlock()
			}
			return
		}
	}
}


func (simulator *Simulator) HandleMessages(txs []*types.Transaction) []error {
	fmt.Println("How many messages  time %s length %d", time.Now(), len(txs))

	// Filter out known ones without obtaining the pool lock or recovering signatures
	var (
		errs = make([]error, len(txs))
		news = make([]*types.Transaction, 0, len(txs))
	)
	for i, tx := range txs {
		// If a transaction has already been executed
		if txres := simulator.simTxPool.executed[tx.Hash()]; txres != nil {
			errs[i] = SimErrAlreadyExecuted
			continue
		}

		// If the transaction is known, pre-set the error slot
		if simulator.simTxPool.all.Get(tx.Hash()) != TxStatusUnknown {
			errs[i] = SimErrAlreadyKnown
			// knownTxMeter.Mark(1)
			continue
		}

		// If the transaction fails basic validation, discard it
		if valerr := simulator.simTxPool.validateTx(tx); valerr != nil {
			errs[i] = valerr
			continue
		}

		// Accumulate all unknown transactions for deeper processing
		news = append(news, tx)
	}
	if len(news) == 0 {
		return errs
	}
	fmt.Printf("How many new transactions  time %s length %d", time.Now(), len(news))

	// Process all the new transaction and merge any errors into the original slice
	fmt.Println("test0")
	simulator.simTxPool.mu.Lock()
	fmt.Println("test1")
	newErrs := simulator.simTxPool.addTxsLocked(news)
	fmt.Println("test4")
	simulator.simTxPool.mu.Unlock()
	fmt.Println("test6")

	var nilSlot = 0
	for _, err := range newErrs {
		for errs[nilSlot] != nil {
			nilSlot++
		}
		errs[nilSlot] = err
		nilSlot++
	}

	// notify the loop to execute the transactions???????
	simulator.newTxsCh <- news

	return errs

}



// execute one transaction
// Modify from commitTransaction
func (simulator *Simulator) ExecuteTransaction(tx *types.Transaction) ([]*types.Log, error) {
	// trace.SimFlag = true
	fmt.Println("test simulation begin")
	start := time.Now()
	fmt.Printf("ExecuteTransaction start time %s \n", start)
	// fmt.Println("ExecuteTransaction?")
	parent := simulator.chain.CurrentBlock()
	// fmt.Println("ExecuteTransaction??")
	current_state, err := simulator.chain.StateAt(parent.Root())
	current_state = current_state.Copy()
	// fmt.Printf("ExecuteTransaction Curent len of revisions %s %s %d\n", time.Now(), current_state.GetOriginalRoot().String(), len(current_state.GetRevisionList()))
	// fmt.Println("ExecuteTransaction???")
	
	snap := current_state.Snapshot()
	// fmt.Printf("ExecuteTransaction Curent len of revisions %s %s %d\n", time.Now(), current_state.GetOriginalRoot().String(), len(current_state.GetRevisionList()))
	
	// fmt.Println("ExecuteTransaction????")
	// fmt.Println("snap id %d", snap)
	num := parent.Number()
	// fmt.Println("ExecuteTransaction?????")
	// fmt.Printf("ExecuteTransaction Current state obtained \n")
	// fmt.Printf("ExecuteTransaction Tx hash  %s\n", tx.Hash().String())
	// fmt.Printf("ExecuteTransaction Curent number (parent number from the blockchain) %s %d\n", time.Now(), num)
	// fmt.Printf("ExecuteTransaction downloader highest number %s %d\n", time.Now(), simulator.eth.Downloader().Progress().HighestBlock)
	// fmt.Printf("ExecuteTransaction downloader current number %s %d\n", time.Now(), simulator.eth.Downloader().Progress().CurrentBlock)

	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		// GasLimit:   core.CalcGasLimit(parent, e.config.GasFloor, e.config.GasCeil),
		GasLimit:   10000000000,
		// Extra:      e.extra,
		Extra:		nil,
		Time:       uint64(time.Now().Unix()),
		// for now
		Difficulty:  big.NewInt(0),
	}
	// fmt.Println("ExecuteTransaction??????")
	// fmt.Printf("ExecuteTransaction header time now  %s\n", time.Now())

	gasPool := new(core.GasPool).AddGas(header.GasLimit)
	// gasPool := new(core.GasPool).AddGas(math.MaxUint64)

	// fmt.Printf("Start RTApplyTransaction\n")
	receipt, err := core.RTApplyTransaction(simulator.chainConfig, simulator.chain, nil, gasPool, current_state, header, tx, &header.GasUsed, *simulator.chain.GetVMConfig())
	// fmt.Printf("End RTApplyTransaction\n")

	
	// fmt.Printf("ExecuteTransaction current parent num %s %d\n", time.Now(), simulator.chain.CurrentBlock().Number())
	// fmt.Printf("ExecuteTransaction Curent len of revisions %s %s %d\n", time.Now(), current_state.GetOriginalRoot().String(), len(current_state.GetRevisionList()))
	current_state.RevertToSnapshot(snap)
	// fmt.Printf("ExecuteTransaction after revert time %s \n", time.Now())

	if err != nil {
		fmt.Println("core.RTApplyTransaction error ", err.Error())
		return nil, err
	} 

	// fmt.Printf("ExecuteTransaction end time %s \n", time.Now())
	fmt.Println("execution duration ", time.Since(start))
	// fmt.Println("test simulation end")
	// trace.SimFlag = false
	// os.Exit(1)
	return receipt.Logs, nil
}




var (
	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	SimErrAlreadyKnown = errors.New("already known")

	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	SimErrAlreadyExecuted = errors.New("already executed")

	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	SimErrInvalidSender = errors.New("invalid sender")

	// ErrUnderpriced is returned if a transaction's gas price is below the minimum
	// configured for the transaction pool.
	SimErrUnderpriced = errors.New("transaction underpriced")

	// ErrTxPoolOverflow is returned if the transaction pool is full and can't accpet
	// another remote transaction.
	SimErrTxPoolOverflow = errors.New("txpool is full")

	// ErrReplaceUnderpriced is returned if a transaction is attempted to be replaced
	// with a different one without the required price bump.
	SimErrReplaceUnderpriced = errors.New("replacement transaction underpriced")

	// ErrGasLimit is returned if a transaction's requested gas limit exceeds the
	// maximum allowance of the current block.
	SimErrGasLimit = errors.New("exceeds block gas limit")

	// ErrNegativeValue is a sanity error to ensure no one is able to specify a
	// transaction with a negative value.
	SimErrNegativeValue = errors.New("negative value")

	// ErrOversizedData is returned if the input data of a transaction is greater
	// than some meaningful limit a user might use. This is not a consensus error
	// making the transaction invalid, rather a DOS protection.
	SimErrOversizedData = errors.New("oversized data")

	// ErrNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	SimErrNonceTooLow = errors.New("nonce too low")

	// ErrInsufficientFundsForTransfer is returned if the transaction sender doesn't
	// have enough funds for transfer(topmost call only).
	SimErrInsufficientFundsForTransfer = errors.New("insufficient funds for transfer")

	// ErrIntrinsicGas is returned if the transaction is specified to use less gas
	// than required to start the invocation.
	SimErrIntrinsicGas = errors.New("intrinsic gas too low")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	SimErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")
)


type SimTxPool struct {
	// general info
	// config      TxPoolConfig
	chainconfig *params.ChainConfig
	chain       *core.BlockChain
	signer      types.Signer


	// pending transactions are ones to be executed
	pending  map[common.Hash]*types.Transaction
	// queue transactions have not reached requirements, like the message nonce, 
	queue  map[common.Hash]*types.Transaction
	// already been executed?
	// In this level, excludes the transactions have been executed
	executed map[common.Hash]*types.Transaction
	// all the tx in pool
	// find whether in the pending or the queue, or the executed
	all     *txLookup                    // All transactions to allow lookup

	// add pool
	mu          sync.RWMutex

	// Current gas limit for transaction caps
	currentMaxGas uint64       

	// the current state to check the transaction might be different from the execution environment???????????????????????????????????????
	// currentState  *state.StateDB // Current state in the blockchain head  
}



func NewSimTxPool(chainconfig *params.ChainConfig, chain *core.BlockChain) *SimTxPool {
	// Create the transaction pool with its initial settings
	pool := &SimTxPool{
		// config:          config,
		chainconfig:     chainconfig,
		chain:           chain,
		signer:          types.NewEIP155Signer(chainconfig.ChainID),

		pending:         make(map[common.Hash]*types.Transaction),
		queue:           make(map[common.Hash]*types.Transaction),
		executed: 		 make(map[common.Hash]*types.Transaction),
		all:             newTxLookup(),

		currentMaxGas:   chain.CurrentBlock().Header().GasLimit, 
	}

	return pool
}


const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// txSlotSize is used to calculate how many data slots a single transaction
	// takes up based on its size. The slots are used as DoS protection, ensuring
	// that validating a new transaction remains a constant operation (in reality
	// O(maxslots), where max slots are 4 currently).
	txSlotSize = 32 * 1024

	// txMaxSize is the maximum size a single transaction can have. This field has
	// non-trivial consequences: larger transactions are significantly harder and
	// more expensive to propagate; larger transactions also take more resources
	// to validate whether they fit into the pool or not.
	txMaxSize = 4 * txSlotSize // 128KB
)



// validateTx checks whether a transaction is valid according to the consensus
// rules and adheres to some heuristic limits of the local node (price and size).
func (pool *SimTxPool) validateTx(tx *types.Transaction) error {
	// Reject transactions over defined size to prevent DOS attacks
	if uint64(tx.Size()) > txMaxSize {
		return SimErrOversizedData
	}
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return SimErrNegativeValue
	}
	// Ensure the transaction doesn't exceed the current block limit gas.
	if pool.currentMaxGas < tx.Gas() {
		return SimErrGasLimit
	}
	// Make sure the transaction is signed properly
	from, err := types.Sender(pool.signer, tx)
	if err != nil {
		return SimErrInvalidSender
	}

	// // Drop non-local transactions under our own minimal accepted gas price
	// if !local && tx.GasPriceIntCmp(pool.gasPrice) < 0 {
	// 	return ErrUnderpriced
	// }
	current_block := pool.chain.CurrentBlock()
	current_state, _ := pool.chain.StateAt(current_block.Root())
	current_state = current_state.Copy()
	// Ensure the transaction adheres to nonce ordering
	if current_state.GetNonce(from) > tx.Nonce() {
		return SimErrNonceTooLow
	}
	fmt.Println("validateTx time %s tx hash %s block %d from %s nonce %d, txnonce %d", time.Now(), tx.Hash(), current_block.Number(), from, current_state.GetNonce(from), tx.Nonce())

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if current_state.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return SimErrInsufficientFunds
	}
	// Ensure the transaction has more gas than the basic tx fee.
	istanbul := pool.chainconfig.IsIstanbul(current_block.Number())
	intrGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true, istanbul)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return SimErrIntrinsicGas
	}
	return nil
}



// addTxsLocked attempts to queue a batch of transactions if they are valid.
// The transaction pool lock must be held.
func (pool *SimTxPool) addTxsLocked(txs []*types.Transaction) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		// already validated
		from, _ := types.Sender(pool.signer, tx)
		// should be listed in the queue?
		current_block := pool.chain.CurrentBlock()
		current_state, _ := pool.chain.StateAt(current_block.Root())
		current_state = current_state.Copy()
		fmt.Println("addTxs time %s tx hash %s block %d from %s nonce %d, txnonce %d", time.Now(), tx.Hash(), current_block.Number(), from, current_state.GetNonce(from), tx.Nonce())

		if current_state.GetNonce(from) < tx.Nonce() {
			pool.queue[tx.Hash()] = tx
			pool.all.Add(tx, false)
			continue
		}
		pool.pending[tx.Hash()] = tx
		pool.all.Add(tx, true)
		errs[i] = nil
	}
	return errs
}


// txLookup is used internally by TxPool to track transactions while allowing
// lookup without mutex contention.
//
// Note, although this type is properly protected against concurrent access, it
// is **not** a type that should ever be mutated or even exposed outside of the
// transaction pool, since its internal state is tightly coupled with the pools
// internal mechanisms. The sole purpose of the type is to permit out-of-bound
// peeking into the pool in TxPool.Get without having to acquire the widely scoped
// TxPool.mu mutex.
type txLookup struct {
	// Mia add: do not understand the slots
	// slots   int
	lock     sync.RWMutex
	pending  map[common.Hash]*types.Transaction
	queue    map[common.Hash]*types.Transaction
}


// newTxLookup returns a new txLookup structure.
func newTxLookup() *txLookup {
	return &txLookup{
		pending:  make(map[common.Hash]*types.Transaction),
		queue:    make(map[common.Hash]*types.Transaction),
	}
}


type TxStatus uint

const (
	TxStatusUnknown TxStatus = iota
	TxStatusQueued
	TxStatusPending
)

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) Get(hash common.Hash) TxStatus {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if tx := t.pending[hash]; tx != nil {
		return TxStatusPending
	}

	if tx := t.queue[hash]; tx != nil {
		return TxStatusQueued
	}

	return TxStatusUnknown
}


// Add adds a transaction to the lookup.
func (t *txLookup) Add(tx *types.Transaction, pending bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// t.slots += numSlots(tx)
	// slotsGauge.Update(int64(t.slots))

	if pending {
		t.pending[tx.Hash()] = tx
	} else {
		t.queue[tx.Hash()] = tx
	}
}


// Remove removes a transaction from the lookup.
func (t *txLookup) Remove(hash common.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	_, ok := t.pending[hash]
	if !ok {
		_, ok = t.queue[hash]
	}
	if !ok {
		log.Error("No transaction found to be deleted", "hash", hash)
		return
	}
	// t.slots -= numSlots(tx)
	// slotsGauge.Update(int64(t.slots))

	delete(t.pending, hash)
	delete(t.queue, hash)
}
