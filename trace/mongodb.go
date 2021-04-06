package trace

import (
	// "bytes"
	"os"
	"gopkg.in/mgo.v2"
)

var SessionGlobal *mgo.Session
var CurrentTx string
var CurrentBlockNum uint64
var TxVMErr string
var ErrorFile *os.File
var DBAll *mgo.Collection

func InitMongoDb() {
	var err error
    	if SessionGlobal, err = mgo.Dial(""); err != nil {
        	panic(err)
   	}

	ErrorFile, err = os.OpenFile("db_error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	DBAll = SessionGlobal.DB("project2_new").C("info")
}


var BashNum int = 100
var BashTxs = make([]interface{}, BashNum)
var CurrentNum int = 0
var Round int = 0


var RTSessionGlobal *mgo.Session
var RTErrorFile *os.File
var Realtime *mgo.Collection

func InitRealtimeDB() {
	var err error
    	if RTSessionGlobal, err = mgo.Dial(""); err != nil {
        	panic(err)
   	}

	RTErrorFile, err = os.OpenFile("realtime.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	Realtime = RTSessionGlobal.DB("project2_new").C("realtime")
}


