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
	"math/big"
	"github.com/ethereum/go-ethereum/trace"
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
	fmt.Println("ExecuteTransaction?")
	parent := simulator.chain.CurrentBlock()
	fmt.Println("ExecuteTransaction??")
	current_state, err := simulator.chain.StateAt(parent.Root())
	fmt.Println("ExecuteTransaction???")
	trace.SimFlag = true
	snap := current_state.Snapshot()
	trace.SimFlag = false
	fmt.Println("ExecuteTransaction????")
	fmt.Println("snap id %d", snap)
	num := parent.Number()
	fmt.Println("ExecuteTransaction?????")
	fmt.Printf("ExecuteTransaction Current state obtained \n")
	fmt.Printf("ExecuteTransaction Parent number %d", num, "\n")
	fmt.Printf("ExecuteTransaction Tx hash  %s\n", tx.Hash().String())

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
	fmt.Println("ExecuteTransaction??????")
	fmt.Printf("ExecuteTransaction header time now  %s\n", time.Now())

	gasPool := new(core.GasPool).AddGas(header.GasLimit)
	fmt.Printf("Start RTApplyTransaction\n")
	receipt, err := core.RTApplyTransaction(simulator.chainConfig, simulator.chain, nil, gasPool, current_state, header, tx, &header.GasUsed, *simulator.chain.GetVMConfig())
	fmt.Printf("End RTApplyTransaction\n")

	trace.SimFlag = true
	fmt.Println("ExecuteTransaction current parent num %d", simulator.chain.CurrentBlock().Number())
	current_state.RevertToSnapshot(snap)
	trace.SimFlag = false

	if err != nil {
		return nil, err
	} 

	fmt.Println("during for a simulation ", time.Since(start))
	fmt.Println("test simulation end")
	os.Exit(1)
	return receipt.Logs, nil
}