package realtime

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"os"
)

// Backend wraps all methods required for mining.
type Backend interface {
	BlockChain() *core.BlockChain
	TxPool() *core.TxPool
}

// Miner creates blocks and searches for proof-of-work values.
// Simulates Miner, Simulator just executes the realtime transactions.
type Simulator struct {
	// executor   *executor  // to execute the transaction
	// coinbase common.Address
	// eth      Backend
	// engine   consensus.Engine
	// exitCh   chan struct{}
	// startCh  chan struct{}
	// startCh  chan common.Address
	// stopCh   chan struct{}

	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain
}

func New(eth Backend, chainConfig *params.ChainConfig, engine consensus.Engine) *Simulator {
	simulator := &Simulator{
		chainConfig:        chainConfig,
		engine:             engine,
		eth:                eth,
		chain:              eth.BlockChain(),		
		// exitCh:  make(chan struct{}),
		// startCh: make(chan struct{}),
		// stopCh:  make(chan struct{}),
		// executor:  newExecutor(chainConfig, engine, eth),
	}
	// go simulator.update()

	return simulator
}


// update keeps track of the downloader events. Please be aware that this is a one shot type of update loop.
// It's entered once and as soon as `Done` or `Failed` has been broadcasted the events are unregistered and
// the loop is exited. This to prevent a major security vuln where external parties can DOS you with blocks
// and halt your mining operation for as long as the DOS continues.

// What this update function should do?????? in the mining 
// func (simulator *Simulator) update() {
// 	// shouldStart := false
// 	canStart := true
// 	for {
// 		select {
// 		case <-simulator.startCh:
// 			if canStart {
// 				simulator.executor.start()
// 			}
// 			// shouldStart = true
// 		case <-simulator.stopCh:
// 			// shouldStart = falses
// 			simulator.executor.stop()
// 		case <-simulator.exitCh:
// 			simulator.executor.close()
// 			return
// 		}
// 	}
// }


// func (simulator *Simulator) Start() {
// 	simulator.startCh <- struct{}{}
// }

// func (simulator *Simulator) Stop() {
// 	simulator.stopCh <- struct{}{}
// }

// func (simulator *Simulator) Close() {
// 	close(simulator.exitCh)
// }

// func (simulator *Simulator) Execute(tx *types.Transaction) {
// 	fmt.Println("Execute func begin")
// 	simulator.executor.executeTransaction(tx)
// 	fmt.Println("Execute func end")
// }



// execute one transaction
// Modify from commitTransaction
func (simulator *Simulator) ExecuteTransaction(tx *types.Transaction) ([]*types.Log, error) {
	fmt.Println("test simulation begin")
	start := time.Now()
	fmt.Println("?")
	parent := simulator.chain.CurrentBlock()
	fmt.Println("??")
	current_state, err := simulator.chain.StateAt(parent.Root())
	fmt.Println("???")
	snap := current_state.Snapshot()
	fmt.Println("????")
	num := parent.Number()
	fmt.Println("?????")
	fmt.Printf("Current state obtained \n")
	fmt.Printf("Parent number %d", num, "\n")
	fmt.Printf("Tx hash  %s\n", tx.Hash().String())

	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     num.Add(num, common.Big1),
		// GasLimit:   core.CalcGasLimit(parent, e.config.GasFloor, e.config.GasCeil),
		GasLimit:   10000000000,
		// Extra:      e.extra,
		Extra:		nil,
		Time:       uint64(time.Now().Unix()),
	}
	fmt.Println("??????")
	fmt.Printf("header time now  %s\n", time.Now())

	gasPool := new(core.GasPool).AddGas(header.GasLimit)
	fmt.Printf("*******************Start RTApplyTransaction**********************\n")
	receipt, err := core.RTApplyTransaction(simulator.chainConfig, simulator.chain, nil, gasPool, current_state, header, tx, &header.GasUsed, *simulator.chain.GetVMConfig())
	fmt.Printf("********************End RTApplyTransaction***********************\n")
	current_state.RevertToSnapshot(snap)

	if err != nil {
		return nil, err
	} 

	fmt.Println("during for a simulation ", time.Since(start))
	fmt.Println("test simulation end")
	os.Exit(1)
	return receipt.Logs, nil
}