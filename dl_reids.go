package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/gosexy/redis"
	"sync"
)

type Dl_Redis struct {
	host string
	port uint
	db int
	key string
	_pdb *redis.Client
	_mtx  *sync.Mutex
}

const (
	SFX_RUNNING = "_RUNNING"
	SFX_SUCCESS = "_SUCCESS"
	SFX_FAILED  = "_FAILED"
)

func NewRedisDl(host string, port uint, password string, db int, key string) *Dl_Redis {
	//log.Debugf("NewRedisDl: host=%s, port=%d, password=%s, db=%d, key=%s \n", host, port, password, db, key)
	dl := new(Dl_Redis)
	dl.host = host
	dl.port = port
	dl.db = db
	dl.key = key
	dl._pdb = redis.New()
	dl._mtx = &sync.Mutex{}
	err := dl._pdb.Connect(host, port)
	if nil != err {
		log.Errorln("NewRedisDl: failed >", err)
		return nil
	}
	if password != "" {
		dl._pdb.Auth(password)
	}
	if db > 0 {
		dl._pdb.Select(int64(db))
	}
	return dl
}

func (self *Dl_Redis) NextLinks(limit int) ([]string, error) {
	ret := []string{}
	self._mtx.Lock()
	for i:=0; i< limit; i++ {
		l, e := self._pdb.LPop(self.key)
		if nil == e {
			ret = append(ret, l)
			self._pdb.HSet(self.key+SFX_RUNNING, l, 1)
		}else{
			log.Errorln("Dl_Redis.NextLinks> failed:", e)
			break
		}
	}
	self._mtx.Unlock()
	return ret, nil
}

func (self *Dl_Redis) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
	self._mtx.Lock()
	ret := map[string]interface{}{"link":link, "dest": dest, "ret_code": ret_code, "ret_msg": ret_msg}
	if ret_code != 0 {
		self._pdb.HSet(self.key+SFX_FAILED, link, ret)
	}else {
		self._pdb.HSet(self.key+SFX_SUCCESS, link, ret)
	}
	self._pdb.HDel(self.key+SFX_RUNNING, link)
	self._mtx.Unlock()
}

