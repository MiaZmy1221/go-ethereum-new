package trace

import (
	// "bytes"
	// "os"
	// "gopkg.in/mgo.v2"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	// "math"
	"math/big"
	"encoding/hex"
)


type TraceN struct{
	CallType string `json:"callType"`
	FromAddr common.Address `json:"fromAddr"`
	ToAddr common.Address `json:"toAddr"`
	Input string `json:"input"`
	Output []byte `json:"output"`
	Value *big.Int `json:"value"`
	TraceIndex uint64 `json:"traceIndex"`
	Type string `json:"type"`
}

func NewTraceN(CallType string, FromAddr common.Address, ToAddr common.Address, Input []byte, Value *big.Int, TraceIndex uint64, Type string, Output []byte) *TraceN {
	t := &TraceN{}
	t.CallType = CallType
	t.FromAddr = FromAddr
	t.ToAddr = ToAddr
	t.Input = hex.EncodeToString(Input)
	t.Value = Value
	t.TraceIndex = TraceIndex
	t.Type = Type
	t.Output = Output
	return t
} 

// Print dumps the content of the memory.
func (t *TraceN) Print() {
	fmt.Printf("### Trace ###\n")
	fmt.Printf("TraceIndex: %d\n", t.TraceIndex)
	fmt.Printf("Call type: %s\n", t.CallType)
	fmt.Printf("From: %s\n", t.FromAddr)
	fmt.Printf("To: %s\n", t.ToAddr)
	fmt.Printf("Input: %s\n", t.Input)
	fmt.Printf("Value: %d\n", t.Value)
	fmt.Printf("Type: %s\n", t.Type) 
	fmt.Printf("Output: %x\n", t.Output) 
	fmt.Println("####################")
}


type TraceNs []*TraceN
var CurrentTraceIndex = uint64(1)
var Traces TraceNs


