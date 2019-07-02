package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/go-redis/redis"
	"sync"
)

type Dl_Redis struct {
	host string
	port uint
	db   int
	key  string
	_pdb *redis.Client
	_mtx sync.Mutex
}

const (
	SfxRunning = "_RUNNING"
	SfxSuccess = "_SUCCESS"
	SfxFailed  = "_FAILED"
)

func NewRedisDl(host string, port uint, password string, db int, key string) *Dl_Redis {
	//log.Debugf("NewRedisDl: host=%s, port=%d, password=%s, db=%d, key=%s \n", host, port, password, db, key)
	dl := new(Dl_Redis)
	dl.host = host
	dl.port = port
	dl.db = db
	dl.key = key
	//dl._pdb = redis.New()
	dl._pdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		DB:       int(db),
		Password: password,
	})
	dl._mtx = sync.Mutex{}
	//err := dl._pdb.Connect(host, port)
	//if nil != err {
	//    log.Errorln("NewRedisDl: failed >", err)
	//    return nil
	//}
	//if password != "" {
	//    dl._pdb.Auth(password)
	//}
	//if db > 0 {
	//    dl._pdb.Select(int64(db))
	//}
	return dl
}

func (dl_redis *Dl_Redis) NextLinks(limit int) ([]string, error) {
	var ret []string
	dl_redis._mtx.Lock()
	for i := 0; i < limit; i++ {
		l, e := dl_redis._pdb.LPop(dl_redis.key).Result()
		if nil == e {
			ret = append(ret, l)
			if _, e := dl_redis._pdb.HSet(dl_redis.key+SfxRunning, l, 1).Result(); nil != e {
				return nil, e
			}
		} else {
			log.Errorln("Dl_Redis.NextLinks> failed:", e)
			break
		}
	}
	dl_redis._mtx.Unlock()
	return ret, nil
}

func (dl_redis *Dl_Redis) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
	dl_redis._mtx.Lock()
	ret := map[string]interface{}{"link": link, "dest": dest, "ret_code": ret_code, "ret_msg": ret_msg}
	if ret_code != 0 {
		if _, e := dl_redis._pdb.HSet(dl_redis.key+SfxFailed, link, ret).Result(); nil != e {
			log.Errorln("Dl_Redis.SubmitResult> failed:", e)
		}
	} else {
		if _, e := dl_redis._pdb.HSet(dl_redis.key+SfxSuccess, link, ret).Result(); nil != e {
			log.Errorln("Dl_Redis.SubmitResult> failed:", e)
		}
	}
	_,_ = dl_redis._pdb.HDel(dl_redis.key+SfxRunning, link).Result()
	dl_redis._mtx.Unlock()
}
