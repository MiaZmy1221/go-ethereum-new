package trace

import (
	// "bytes"
	// "os"
	// "gopkg.in/mgo.v2"
	"fmt"
	// "github.com/ethereum/go-ethereum/common"
	// "math"
	"math/big"
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
	fmt.Printf("Input: %s\n", t.Input)
	fmt.Printf("Value: %d\n", t.Value)
	fmt.Printf("Type: %s\n", t.Type) 
	fmt.Printf("Output: %s\n", t.Output) 
	fmt.Println("####################")
}


// type TraceNs []*TraceN
// var Traces TraceNs

var Traces = []TraceN{}
var CurrentTraceIndex = uint64(1)



