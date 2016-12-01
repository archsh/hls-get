package main

import (
	log "github.com/Sirupsen/logrus"
	"strings"
)

type DL_Interface interface {
	NextLinks(limit int) ([]string, error)
	SubmitResult(link string, dest string, ret_code int, ret_msg string)
}


type Dl_Dummy struct {
	links []string
	cursor int
}

func NewDummyDl(links []string) *Dl_Dummy {
	dl := new(Dl_Dummy)
	for _, l := range links {
		dl.links = append(dl.links, strings.TrimSpace(l))
	}
	dl.links = links
	return dl
}

func (self *Dl_Dummy) NextLinks(limit int) ([]string, error) {
	//log.Debugln("Dl_Dummy.NextLinks>", self.cursor, limit, len(self.links))
	if self.cursor >= len(self.links){
		return []string{}, nil
	}else{
		var ret []string
		if self.cursor + limit > len(self.links) {
			//log.Debugln("Dl_Dummy.NextLinks> P1 ")
			ret = self.links[self.cursor:]
		}else{
			//log.Debugln("Dl_Dummy.NextLinks> P2")
			ret = self.links[self.cursor:self.cursor+limit]
		}
		self.cursor += limit
		return ret, nil
	}
}

func (self *Dl_Dummy) SubmitResult(link string, dest string, ret_code int, ret_msg string) {
	log.Infoln("DL >", link, dest, ret_code, ret_msg)
}
