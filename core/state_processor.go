// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/trace"
	"encoding/json"
	"encoding/hex"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	blockContext := NewEVMBlockContext(header, p.bc, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := tx.AsMessage(types.MakeSigner(p.config, header.Number))
		if err != nil {
			return nil, nil, 0, err
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, err := applyTransaction(msg, p.config, p.bc, nil, gp, statedb, header, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}

	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles())

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// # Step1: the first trace currently unknown output
	fmt.Printf("state_processor.go applyTransaction\n")
	fmt.Printf("\nThe first trace \n")
	fmt.Printf("Call type: CALL\n")
	fmt.Printf("From: %s\n", msg.From())
	fmt.Printf("To: %s\n", msg.To())
	fmt.Printf("Create \n")
	fmt.Printf("Input: 0x%x\n", msg.Data())
	fmt.Printf("Value: %d\n", msg.Value())
	fmt.Printf("TraceIndex: %d\n", 1)
	fmt.Printf("Type: CALL\n") // other types: suicide
	fmt.Printf("Output: %x\n") // ????

	// # Step2: in the interpreter.go
	// Step 2.1 deal with the trace
	// Step 2.2 deal with the event Transfer ERC20


	// Create a new context to be used in the EVM environment
	txContext := NewEVMTxContext(msg)
	// Add addresses to access list if applicable
	if config.IsYoloV3(header.Number) {
		statedb.AddAddressToAccessList(msg.From())
		if dst := msg.To(); dst != nil {
			statedb.AddAddressToAccessList(*dst)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range evm.ActivePrecompiles() {
			statedb.AddAddressToAccessList(addr)
		}
	}

	// Update the evm with the new transaction context.
	evm.Reset(txContext, statedb)
	// Apply the transaction to the current state (included in the env)
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing whether the root touch-delete accounts.
	receipt := types.NewReceipt(root, result.Failed(), *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())

	fmt.Printf("\nFrom state_processor.go applyTransaction\n")
	fmt.Printf("Tx hash %s \n", receipt.TxHash)
	fmt.Printf("Tx index %d \n", uint(statedb.TxIndex()))
	for _, tempt_log := range receipt.Logs{
		fmt.Printf("Tx log topics %s \n", tempt_log.Topics)
		fmt.Printf("Tx log data 0x%x \n", tempt_log.Data)
		fmt.Printf("Tx log address %s \n", tempt_log.Address)
		fmt.Printf("Tx log removed %t \n", tempt_log.Removed)
		fmt.Printf("Tx log blocknum %d \n", tempt_log.BlockNumber)
	}
	fmt.Printf("Tx status %d\n", receipt.Status)


	// # Step1: deal with the first trace's output
	first_trace := &trace.TraceN{
		CallType: "CALL", 
		FromAddr: msg.From().String(), 
		ToAddr: *msg.To().String(), 
		CreateAddr: "0x"
		SuicideContract: "0x"
		Beneficiary: "0x"
		Input: hex.EncodeToString(msg.Data()),
		Output: hex.EncodeToString(result.ReturnData), 
		Value: msg.Value(), 
		TraceIndex: 1, 
		Type: "CALL"}
	json_first_trace, _ := json.Marshal(first_trace)
	fmt.Println(string(json_first_trace))

	var tempt_traces trace.TraceNs
	tempt_traces = append(tempt_traces, first_trace)
	trace.Traces = append(tempt_traces, trace.Traces...)


	// # Step3: print all the traces
	fmt.Printf("*************************In the end, traces are ***********************\n")
	for _, tempt_trace := range trace.Traces{
		tempt_trace.Print()
	}


	
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = statedb.BlockHash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(statedb.TxIndex())

	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	// vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	vmenv := vm.NewEVMWithFlag(blockContext, vm.TxContext{}, statedb, config, cfg, false)
	return applyTransaction(msg, config, bc, author, gp, statedb, header, tx, usedGas, vmenv)
}
