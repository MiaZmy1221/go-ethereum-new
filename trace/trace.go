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


var Traces = []TraceN{}
var CurrentTraceIndex = uint64(0)


type TransferLog struct{
	FromAddr string `json:"fromAddr"`
	ToAddr string `json:"toAddr"`
	Value string `json:"value"`
	TokenAddr string `json:"tokenAddr"`
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

var TransferLogs = []TransferLog{}


type TxReceipt struct {
	BlockNum string 
	FromAddr string
	ToAddr string 
	Gas string
	GasUsed string
	GasPrice string
	TxHash string 
	TxIndex uint
	Value string
	Input string
	Status  string
	Err string
}

var GTxReceipt = &TxReceipt{}
var CreatedSC []string 
// var ErrorFile *os.File


