package realtime

import (
	"os"
	"gopkg.in/mgo.v2"
	// "github.com/ethereum/go-ethereum/core/types"
)

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



