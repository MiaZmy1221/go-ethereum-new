package realtime

import (
	// "bytes"
	// "errors"
	// "math/big"
	// "sync"
	"sync/atomic"
	// "time"

	// mapset "github.com/deckarep/golang-set"
	// "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	// "github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	// "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	// "github.com/ethereum/go-ethereum/event"
	// "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	// "github.com/ethereum/go-ethereum/trie"
)


// worker is the main object which takes care of submitting new work to consensus engine
// and gathering the sealing result.
type executor struct {
	config      *Config
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// startCh            chan struct{}
	exitCh             chan struct{}

	// added 
	newtxCh				chan *types.Transaction

	// The lock used to protect the coinbase
	// mu       sync.RWMutex 
	// coinbase common.Address

	// atomic status counters
	running int32 // The indicator whether the consensus engine is running or not.
}

func newExecutor(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend) *worker {
	executor := &executor{
		config:             config,
		chainConfig:        chainConfig,
		engine:             engine,
		eth:                eth,
		chain:              eth.BlockChain(),
		
		exitCh:             make(chan struct{}),
		newtxCh:			make(chan *types.Transaction),
	}

	go executor.mainLoop()

	return executor
}

// mainLoop is a standalone goroutine to regenerate the sealing task based on the received event.
func (e *executor) mainLoop() {
	for {
		select {
		// a new tx comes
		case new_tx := <-e.newtxCh:
			// make sure the executor is running
			if e.isRunning() {
				// Only execute one transaction for now
				// We do not consider txs relatonship: such as sort by price ... 
				e.executeTransaction(new_tx)	
			} 
		// System stopped
		case <-w.exitCh:
			return
		}
	}
}


// execute one transaction
// Modify from commitTransaction
func (e *executor) executeTransaction(tx *types.Transaction) ([]*types.Log, error) {
	parent := e.chain.CurrentBlock()
	current_state, err := e.chain.StateAt(parent.Root())

	snap := current_state.Snapshot()
	receipt, err := core.RTApplyTransaction(e.chainConfig, e.chain, &coinbase, e.current.gasPool, current_state, e.current.header, tx, &e.current.header.GasUsed, *e.chain.GetVMConfig())
	current_state.RevertToSnapshot(snap)

	if err != nil {
		return nil, err
	} 
	return receipt.logs, nil
}


// start sets the running status as 1
func (e *executor) start() {
	atomic.StoreInt32(&e.running, 1)
}

// stop sets the running status as 0.
func (e *executor) stop() {
	atomic.StoreInt32(&e.running, 0)
}


// isRunning returns an indicator whether executor is running or not.
func (e *executor) isRunning() bool {
	return atomic.LoadInt32(&e.running) == 1
}

// close terminates all background threads maintained by the executor.
// Note the executor does not support being closed multiple times.
func (e *executor) close() {
	// What is this stopPrefetcher???????????
	// if e.current != nil && e.current.state != nil {
	// 	e.current.state.StopPrefetcher()
	// }
	atomic.StoreInt32(&e.running, 0)
	close(e.exitCh)
}