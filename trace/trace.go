package trace

import (
	"bytes"
	"os"
	// "gopkg.in/mgo.v2"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
)


type TraceN struct{
	CallType string
	FromAddr common.Address
	ToAddr common.Address
	Input []byte
	Value *big.Int
	TraceIndex uint64
	Type string
}

func NewTraceN(CallType string, FromAddr common.Address, ToAddr common.Address, Input []byte, Value *big.Int, TraceIndex uint64, Type string) *TraceN {
	t := &TraceN{}
	t.CallType = CallType
	t.FromAddr = FromAddr
	t.ToAddr = ToAddr
	t.Input = Input
	t.Value = Value
	t.TraceIndex = TraceIndex
	t.Type = Type
	return t
} 

var currentTraceIndex := 1
var Traces := []*TraceN


