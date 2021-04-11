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
	"github.com/ethereum/go-ethereum/trace"
	"github.com/ethereum/go-ethereum/eth/downloader"
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

	running		uint64

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
		running:			uint64(0),	
		startCh: 			make(chan struct{}),
		stopCh:  			make(chan struct{}),
		newTxs:				make(chan []*types.Transaction),
	}
	go simulator.loop()

	simulator.startCh <- chan struct{}

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
			atomic.StoreInt32(&simulator.running,0 1)
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
			errs[i] = ErrAlreadyExecuted
			continue
		}

		// If the transaction is known, pre-set the error slot
		if simulator.simTxPool.all.Get(tx.Hash()) != TxStatusUnknown {
			errs[i] = ErrAlreadyKnown
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

	fmt.Println("How many new transactions  time %s length %d", time.Now(), len(news))

	// Process all the new transaction and merge any errors into the original slice
	simulator.simTxPool.mu.Lock()
	newErrs := simulator.simTxPool.addTxsLocked(news, local)
	simulator.simTxPool.mu.Unlock()

	var nilSlot = 0
	for _, err := range newErrs {
		for errs[nilSlot] != nil {
			nilSlot++
		}
		errs[nilSlot] = err
		nilSlot++
	}

	// notify the loop to execute the transactions???????
	simulator.startCh <- news

	return errs

}



// execute one transaction
// Modify from commitTransaction
func (simulator *Simulator) ExecuteTransaction(tx *types.Transaction) ([]*types.Log, error) {
	trace.SimFlag = true
	fmt.Println("test simulation begin")
	start := time.Now()
	fmt.Printf("ExecuteTransaction start time %s \n", start)
	// fmt.Println("ExecuteTransaction?")
	parent := simulator.chain.CurrentBlock()
	// fmt.Println("ExecuteTransaction??")
	current_state, err := simulator.chain.StateAt(parent.Root())
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
	trace.SimFlag = false
	// os.Exit(1)
	return receipt.Logs, nil
}