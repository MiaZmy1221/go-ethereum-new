package realtime

// import "C"

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
	"github.com/ethereum/go-ethereum/core/state"
	// "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/event"
)

// Backend wraps all methods required for mining.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
	Downloader() *downloader.Downloader
}

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// 
	MAXExecutedNum = 1000

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

	// to pretect the execution, currently used for now
	// exe          sync.RWMutex


	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription

}

// New the simulator, do not worry 
func New(eth Backend, chainConfig *params.ChainConfig, engine consensus.Engine) *Simulator {
	fmt.Println("New the simulator")
	simulator := &Simulator{
		chainConfig:        chainConfig,
		engine:             engine,
		eth:                eth,
		chain:              eth.BlockChain(),	
		simTxPool:   		NewSimTxPool(chainConfig, eth.BlockChain()),
		startCh:            make(chan struct{}, 1),
		stopCh:  			make(chan struct{}),

		// case 1.1: does not have buffer now
		newTxsCh:			make(chan []*types.Transaction),

		chainHeadCh:        make(chan core.ChainHeadEvent, chainHeadChanSize),
	}

	simulator.chainHeadSub = simulator.chain.SubscribeChainHeadEvent(simulator.chainHeadCh)


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
			fmt.Printf("%s start the simulator in the loop\n", time.Now())
			atomic.StoreInt32(&simulator.running, 1)
		case <-simulator.stopCh:
			fmt.Printf("%s stop the simulator in the loop\n", time.Now())
			atomic.StoreInt32(&simulator.running, 0)
		case newTxs := <-simulator.newTxsCh:
			// we do not need the lock right now
			fmt.Printf("%s handle coming newTxs in the loop\n", time.Now())
			if simulator.isRunning() == true {
				for _, tx := range newTxs {
					simulator.ExecuteTransaction(tx)
					simulator.simTxPool.RemoveExecuted(tx)
				}
				fmt.Printf("%s finished execution newTxs coming in the loop length %d\n", time.Now(), len(newTxs))
			}
			fmt.Println("\n\n\n\n\n\n")

		case head := <-simulator.chainHeadCh:
			fmt.Printf("%s new mined block number in the loop %d\n", time.Now(), head.Block.NumberU64())
			// get highest head  if not euqal, continue, if equal, promote
			if head.Block.NumberU64() == simulator.eth.Downloader().Progress().HighestBlock {
				fmt.Printf("%s before PromoteQueue current block in the loop %d\n", time.Now(), head.Block.NumberU64())
				promoted_txs := simulator.simTxPool.PromoteQueue()
				if simulator.isRunning() == true {
					for _, tx := range promoted_txs {
						simulator.ExecuteTransaction(tx)
						simulator.simTxPool.RemoveExecuted(tx)
					}
					fmt.Printf("%s finished execution promoted_txs coming in the loop length %d\n", time.Now(), len(promoted_txs))
				}
			}
			fmt.Println("\n\n\n\n\n\n")
		}
	}
}


var (
	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	SimErrAlreadyKnown = errors.New("already known")

	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	SimErrAlreadyExecuted = errors.New("already executed")
	SimErrAlreadyMined = errors.New("already mined")

	SimErrFailedBasicVal = errors.New("failed basic validation")

	SimErrToEOA = errors.New("To address is EOA")

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



func (simulator *Simulator) HandleMessages(txs []*types.Transaction) []error {
	fmt.Println("%s begin in the HandleMessages\n", time.Now())

	var (
		errs = make([]error, len(txs))
		news = make([]*types.Transaction, 0, len(txs))
	)
	for i, tx := range txs {
		// Setp 1: If the transaction already exists
		tempt_status := simulator.simTxPool.Get(tx.Hash())
		if tempt_status == TxStatusQueued || tempt_status == TxStatusExecuting {
			errs[i] = SimErrAlreadyKnown
			continue
		}
		if tempt_status == TxStatusExecuted {
			errs[i] = SimErrAlreadyExecuted
			continue
		}

		// Step extra: check whether the transaction has already been mined?????
		if simulator.chain.GetReceiptsByHash(tx.Hash()) != nil {
			errs[i] = SimErrAlreadyMined
			continue
		}

		// Step 2: If the transaction fails basic validation, discard it, cheks whether the nonce too low
		current_state, state_err := simulator.chain.StateAt(simulator.chain.CurrentBlock().Root())
		if state_err != nil {
			fmt.Printf("%s Get current state error \n", time.Now())
			return nil
		}

		if valerr := simulator.simTxPool.validateTx(tx, current_state, simulator.chain.CurrentBlock().Number()); valerr != nil {
			errs[i] = SimErrFailedBasicVal
			errs[i] = valerr
			continue
		}

		// Stpe 3: If the to address is EOA, discard it.
		code := current_state.GetCode(*tx.To())
		if len(code) == 0 {
			errs[i] = SimErrToEOA
			continue
		}

		// Step 5: replaced tx????
		// let us forget the replaced for now,


		// Step 4: If the nonce is too high, add it to the queue
		from, _ := types.Sender(simulator.simTxPool.signer, tx) // already passed in the Step2 ValidateTx
		if current_state.GetNonce(from) < tx.Nonce() { // nonce is too high, add it to the queue
			simulator.simTxPool.Add(tx, 1)
			continue
		}

		// Accumulate all unknown transactions for deeper processing
		news = append(news, tx)
	}

	for i, tempt_err := range errs {
		fmt.Printf("%s txhash %s error %s \n", time.Now(), txs[i].Hash().String(), tempt_err.Error())
	}


	if len(news) == 0 {
		return errs
	}

	for _, newtx := range news{
		simulator.simTxPool.Add(newtx, 0)
	}

	fmt.Printf("%s How many time messages %d new txs %d\n", time.Now(), len(txs), len(news))

	

	// we do not use the pending pool, 
	simulator.newTxsCh <- news

	return errs
}



// execute one transaction
// Modify from commitTransaction
func (simulator *Simulator) ExecuteTransaction(tx *types.Transaction) ([]*types.Log, error) {
	// trace.SimFlag = true
	// fmt.Println("test simulation begin")
	start := time.Now()
	// fmt.Printf("ExecuteTransaction start time %s \n", start)
	// fmt.Println("ExecuteTransaction?")
	parent := simulator.chain.CurrentBlock()
	// fmt.Println("ExecuteTransaction??")
	current_state, err := simulator.chain.StateAt(parent.Root())
	if err != nil {
		fmt.Printf("%s Get current state error \n", time.Now())
	}
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

	// detect the attacks
	// C.SayHello(C.CString("Hello, World"))
	
	// fmt.Printf("ExecuteTransaction current parent num %s %d\n", time.Now(), simulator.chain.CurrentBlock().Number())
	// fmt.Printf("ExecuteTransaction Curent len of revisions %s %s %d\n", time.Now(), current_state.GetOriginalRoot().String(), len(current_state.GetRevisionList()))
	current_state.RevertToSnapshot(snap)
	// fmt.Printf("ExecuteTransaction after revert time %s \n", time.Now())

	if err != nil {
		fmt.Println("core.RTApplyTransaction error ", err.Error())
		return nil, err
	} 

	// fmt.Printf("ExecuteTransaction end time %s \n", time.Now())
	fmt.Printf("time %s tx hash %s execution time %s  current number %d\n", time.Now(), tx.Hash().String(), time.Since(start), num)
	// fmt.Println("test simulation end")
	// trace.SimFlag = false
	// os.Exit(1)
	return receipt.Logs, nil
}


type SimTxPool struct {
	// general info
	// config      TxPoolConfig
	chainconfig *params.ChainConfig
	chain       *core.BlockChain
	signer      types.Signer


	// case 1: remove the pending for now
	// pending transactions are ones to be executed
	// pending  map[common.Hash]*types.Transaction

	// queue transactions have not reached requirements, like the message nonce, 
	// queue and executed should be trucated from time to time
	executedList  []common.Hash

	queue  map[common.Hash]*types.Transaction
	executed  map[common.Hash]*types.Transaction
	// current executing transactions (in the executing phase)
	// just the messages related newTxs
	currentExecuting map[common.Hash]*types.Transaction
	lock          sync.RWMutex // to protect all


	// sort by the from address
	// sortedQueue   map[common.Address][]*types.Transaction
	// queLock       sync.RWMutex // to protect sortedQueue

	// Current gas limit for transaction caps
	currentMaxGas uint64 
}



func NewSimTxPool(chainconfig *params.ChainConfig, chain *core.BlockChain) *SimTxPool {
	// Create the transaction pool with its initial settings
	pool := &SimTxPool{
		// config:          config,
		chainconfig:     chainconfig,
		chain:           chain,
		signer:          types.NewEIP155Signer(chainconfig.ChainID),

		currentExecuting:         make(map[common.Hash]*types.Transaction),
		queue:           make(map[common.Hash]*types.Transaction),
		executed: 		 make(map[common.Hash]*types.Transaction),

		// executedList: 		 make([]common.Hash)

		currentMaxGas:   chain.CurrentBlock().Header().GasLimit, 
	}

	

	return pool
}


type TxStatus uint

const (
	TxStatusUnknown TxStatus = iota
	TxStatusExecuting
	TxStatusQueued
	TxStatusExecuted
)


// Get returns a transaction if it exists in the lookup, or nil if not found.
func (pool *SimTxPool) Get(hash common.Hash) TxStatus {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	if tx := pool.currentExecuting[hash]; tx != nil {
		return TxStatusExecuting
	}

	if tx := pool.queue[hash]; tx != nil {
		return TxStatusQueued
	}

	if tx := pool.executed[hash]; tx != nil {
		return TxStatusExecuted
	}

	return TxStatusUnknown
}


// Add adds a transaction to the lookup.
func (pool *SimTxPool) Add(tx *types.Transaction, status int) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if status == 0 {
		pool.currentExecuting[tx.Hash()] = tx
	}

	if status == 1 {
		pool.queue[tx.Hash()] = tx
	}
}


// Remove a transaction from the executing
func (pool *SimTxPool) RemoveExecuted(tx *types.Transaction) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	delete(pool.currentExecuting, tx.Hash())

	// add the executed to the executed map and list
	if len(pool.executed) == MAXExecutedNum {
		removed_hash := pool.executedList[0]
		pool.executedList = pool.executedList[1:]
		pool.executedList = append(pool.executedList, tx.Hash())
		delete(pool.executed, removed_hash)
		pool.executed[tx.Hash()] = tx

	} else {
		pool.executed[tx.Hash()] = tx
		pool.executedList = append(pool.executedList, tx.Hash())
	}
	
}


// Promote transactions
func (pool *SimTxPool) PromoteQueue() []*types.Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	news := []*types.Transaction{}

	// check the queue one by one 
	current_block := pool.chain.CurrentBlock()
	fmt.Printf("PromoteQueue current block %d\n", current_block.Number())
	current_state, err := pool.chain.StateAt(current_block.Root())
	if err != nil {
		fmt.Printf("%s Get current state error\n", time.Now())
		return nil
	}

	for _, tx := range pool.queue {
		from, _ := types.Sender(pool.signer, tx)
		if tx.Nonce() == current_state.GetNonce(from) {
			news = append(news, tx)
		}
		if tx.Nonce() < current_state.GetNonce(from) {
			delete(pool.queue, tx.Hash())
		}
	}
	return news
	
}


// validateTx checks whether a transaction is valid according to the consensus
// rules and adheres to some heuristic limits of the local node (price and size).
func (pool *SimTxPool) validateTx(tx *types.Transaction, current_state *state.StateDB, cbn *big.Int) error {
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

	// Ensure the transaction adheres to nonce ordering
	if current_state.GetNonce(from) > tx.Nonce() {
		return SimErrNonceTooLow
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if current_state.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return SimErrInsufficientFunds
	}
	// Ensure the transaction has more gas than the basic tx fee.
	istanbul := pool.chainconfig.IsIstanbul(cbn)
	intrGas, err := core.IntrinsicGas(tx.Data(), tx.To() == nil, true, istanbul)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return SimErrIntrinsicGas
	}
	return nil
}
