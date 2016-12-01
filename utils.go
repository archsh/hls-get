package main

import (
	//"bytes"
	//"flag"
	//"fmt"
	//"github.com/golang/groupcache/lru"
	//"github.com/gosexy/redis"
	//"github.com/kz26/m3u8"
	//"io"
	//"io/ioutil"
	//log "github.com/Sirupsen/logrus"
	//"net/http"
	//"net/url"
	//"os"
	//"path"
	//"regexp"
	//"strconv"
	//"strings"
	//"time"
	//"github.com/archsh/hlsutils/helpers/logging"
	"github.com/rwtodd/sed-go"
	"strings"
	"os"
)


type StringRewriter interface {
	RunString(string) string
}


type PathRewriter struct {
	engine *sed.Engine
}

func (self *PathRewriter) RunString(input string) string {
	if nil != self.engine {
		s, e := self.engine.RunString(input)
		if nil != e {
			return input
		}else{
			return strings.TrimSpace(s)
		}
	}else{
		return input
	}
}

func NewPathRewriter(cmd string) (pr *PathRewriter) {
	pr = new(PathRewriter)
	if cmd == "" {
		return pr
	}
	engine, err := sed.New(strings.NewReader(cmd))
	if nil == err {
		pr.engine = engine
	}
	return
}

type SegmentRewriter struct {

}

func (self *SegmentRewriter) RunString(input string) string {
	return  input
}

func NewSegmentRewriter(cmd string) (sr *SegmentRewriter){
	sr = new(SegmentRewriter)
	return
}

func exists(path string, size int64) bool {
	s, err := os.Stat(path)
	if nil != err || s.Size() < size {
		return false
	}
	return true
}