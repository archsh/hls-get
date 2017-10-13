package main


import (
	"github.com/boltdb/bolt"
	log "github.com/Sirupsen/logrus"
	//"sync"
)

type Dl_Boltdb struct {
	db *bolt.DB
}


func NewBoltDl(dbfile string) *Dl_Boltdb {
	return nil
}

func (self *Dl_Boltdb) NextLinks(limit int) ([]string, error) {
	log.Debugln(">")
	return nil, nil
}

func (self *Dl_Boltdb) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Debugln(">")
}