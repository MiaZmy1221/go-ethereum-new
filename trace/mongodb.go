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

	DBAll = SessionGlobal.DB("project2").C("info")
}

 

// var DB_created_sc = SessionGlobal.DB("project2").C("created_sc")
// var DB_receipt = SessionGlobal.DB("project2").C("receipt")
// var DB_transfer_log = SessionGlobal.DB("project2").C("transfer_log")
// var DB_trace = SessionGlobal.DB("project2").C("trace")

