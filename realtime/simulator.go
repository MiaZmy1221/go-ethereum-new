package realtime

import (
	// "fmt"
	// "math/big"
	// "time"

	// "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	// "github.com/ethereum/go-ethereum/core/state"
	// "github.com/ethereum/go-ethereum/core/types"
	// "github.com/ethereum/go-ethereum/eth/downloader"
	// "github.com/ethereum/go-ethereum/event"
	// "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Backend wraps all methods required for mining.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
}

// Miner creates blocks and searches for proof-of-work values.
// Simulates Miner, Simulator just executes the realtime transactions.
type Simulator struct {
	executor   *executor  // to execute the transaction
	// coinbase common.Address
	eth      Backend
	engine   consensus.Engine
	exitCh   chan struct{}
	startCh  chan struct{}
	// startCh  chan common.Address
	stopCh   chan struct{}
}

func New(eth Backend, chainConfig *params.ChainConfig, engine consensus.Engine) *Simulator {
	simulator := &Simulator{
		eth:     eth,
		engine:  engine,
		exitCh:  make(chan struct{}),
		startCh: make(chan struct{}),
		stopCh:  make(chan struct{}),
		executor:  newExecutor(chainConfig, engine, eth),
	}
	go simulator.update()

	return simulator
}


// update keeps track of the downloader events. Please be aware that this is a one shot type of update loop.
// It's entered once and as soon as `Done` or `Failed` has been broadcasted the events are unregistered and
// the loop is exited. This to prevent a major security vuln where external parties can DOS you with blocks
// and halt your mining operation for as long as the DOS continues.

// What this update function should do?????? in the mining 
func (simulator *Simulator) update() {
	// shouldStart := false
	canStart := true
	for {
		select {
		case <-simulator.startCh:
			if canStart {
				simulator.executor.start()
			}
			// shouldStart = true
		case <-simulator.stopCh:
			// shouldStart = falses
			simulator.executor.stop()
		case <-simulator.exitCh:
			simulator.executor.close()
			return
		}
	}
}


// func (simulator *Simulator) Simulating() bool {
// 	return simulator.executor.isRunning()
// }

func (simulator *Simulator) Start() {
	simulator.startCh <- struct{}{}
}

func (simulator *Simulator) Stop() {
	simulator.stopCh <- struct{}{}
}

func (simulator *Simulator) Close() {
	close(simulator.exitCh)
}