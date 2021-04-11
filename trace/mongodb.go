package trace

import (
	// "bytes"
	"os"
	"gopkg.in/mgo.v2"
)


// Need for the sync
var SessionGlobal *mgo.Session
var CurrentTx string
var CurrentBlockNum uint64
var TxVMErr string
var ErrorFile *os.File
var DBAll *mgo.Collection
var BashNum int = 100
var BashTxs = make([]interface{}, BashNum)
var CurrentNum int = 0
// var Round int = 0

func InitMongoDb() {
	var err error
    	if SessionGlobal, err = mgo.Dial(""); err != nil {
        	panic(err)
   	}

	ErrorFile, err = os.OpenFile("db_error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	DBAll = SessionGlobal.DB("simulation").C("history")
}



// Need for the simulation
var RTSessionGlobal *mgo.Session
var SimErrorFile *os.File
var Realtime *mgo.Collection

var SimBashNum int = 100
var SimBashTxs = make([]interface{}, SimBashNum)
var SimCurrentNum int = 0

func InitRealtimeDB() {
	var err error
    	if RTSessionGlobal, err = mgo.Dial(""); err != nil {
        	panic(err)
   	}

	SimErrorFile, err = os.OpenFile("realtime.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	Realtime = RTSessionGlobal.DB("simulation").C("pending")
}





