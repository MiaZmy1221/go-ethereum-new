package trace

import (
	// "bytes"
	// "os"
	// "gopkg.in/mgo.v2"
	"fmt"
	// "github.com/ethereum/go-ethereum/common"
	// "math"
	"math/big"
	// "github.com/ethereum/go-ethereum/common"
	// "encoding/hex"
)


type TraceN struct{
	CallType string `json:"callType"`
	FromAddr string `json:"fromAddr"`
	ToAddr string `json:"toAddr"`
	CreateAddr string `json:"createAddr"`
	SuicideContract string `json:"suicideContract"`
	Beneficiary string `json:"beneficiary"`
	Input string `json:"input"`
	Output string `json:"output"`
	Value *big.Int `json:"value"`
	CallDepth int `json:"callDepth"`
	CallNum int `json:"callNum"`
	TraceIndex uint64 `json:"traceIndex"`
	Type string `json:"type"`
}


// Print dumps the content of the memory.
func (t *TraceN) Print() {
	fmt.Printf("### Trace ###\n")
	fmt.Printf("TraceIndex: %d\n", t.TraceIndex)
	fmt.Printf("Call type: %s\n", t.CallType)
	fmt.Printf("From: %s\n", t.FromAddr)
	fmt.Printf("To: %s\n", t.ToAddr)
	fmt.Printf("CreateAddr: %s\n", t.CreateAddr)
	fmt.Printf("SuicideContract: %s\n", t.SuicideContract)
	fmt.Printf("Beneficiary: %s\n", t.Beneficiary)
	fmt.Printf("Input: %s\n", t.Input)
	fmt.Printf("Value: %d\n", t.Value)
	fmt.Printf("Type: %s\n", t.Type) 
	fmt.Printf("Output: %s\n", t.Output) 
	fmt.Println("####################")
}


type TransferLog struct{
	FromAddr string `json:"fromAddr"`
	ToAddr string `json:"toAddr"`
	Value string `json:"value"`
	TokenAddr string `json:"tokenAddr"`
	CallDepth int `json:"callDepth"`
	CallNum int `json:"callNum"`
	TraceIndex uint64 `json:"traceIndex"`
}

func (l *TransferLog) Print() {
	fmt.Printf("### ERC20TransferLog ###\n")
	fmt.Printf("TraceIndex: %d\n", l.TraceIndex)
	fmt.Printf("From: %s\n", l.FromAddr)
	fmt.Printf("To: %s\n", l.ToAddr)
	fmt.Printf("Value: %s\n", l.Value)
	fmt.Printf("TokenAddr: %s\n", l.TokenAddr)
	fmt.Println("####################")
}


type TxReceipt struct {
	BlockNum string `json:"blockNum"`
	FromAddr string `json:"fromAddr"`
	ToAddr string `json:"toAddr"`
	Gas string `json:"gas"`
	GasUsed string `json:"gasUsed"`
	GasPrice string `json:"gasPrice"`
	// TxHash string `json:"txHash"`
	TxIndex uint `json:"txIndex"`
	Value string `json:"value"`
	Input string `json:"input"`
	Status  string `json:"status"`
	Err string `json:"err"`
}

type TransactionAll struct{
	TxHash string `json:"txHash"`
	TxReceipt string `json:"txReceipt"`
	TxTransferLogs string `json:"txTransferLogs"`
	TxTraces string `json:"txTraces"`
	TxCreatedSC string `json:"txCreatedSC"`
}


// Needed for the sync: apply transaction
var Traces = []TraceN{}
var CurrentTxIndex = 0
var CurrentTraceIndex = uint64(0)
var TransferLogs = []TransferLog{}
var GTxReceipt = &TxReceipt{}
var CreatedSC []string 
var CallDepth = 0
var CallNum = -1
var SyncFlag = false

// Needed for the simulation: apply transaction
var SimFlag = false
var SimTraces = []TraceN{}
var SimCurrentTxIndex = 0
var SimCurrentTraceIndex = uint64(0)
var SimTransferLogs = []TransferLog{}
var SimGTxReceipt = &TxReceipt{}
var SimCreatedSC []string 
var SimCallDepth = 0
var SimCallNum = -1


// For test
var TestIndex = 0
var OnlyOneTopic = false



