package main

import (
	"fmt"
	"path"

	//"github.com/rwtodd/sed-go"
	"github.com/rwtodd/Go.Sed/sed"
	"os"
	"strings"
)

type StringRewriter interface {
	RunString(string, int) string
}

type PathRewriter struct {
	engine *sed.Engine
}

func (self *PathRewriter) RunString(input string, n int) string {
	if nil != self.engine {
		s, e := self.engine.RunString(input)
		if nil != e {
			return input
		} else {
			return strings.TrimSpace(s)
		}
	} else {
		return path.Base(input)
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
	format string
}

func (self *SegmentRewriter) RunString(input string, n int) string {
	return fmt.Sprintf(self.format, n)
	//return input
}

func NewSegmentRewriter(cmd string) (sr *SegmentRewriter) {
	sr = new(SegmentRewriter)
	sr.format = cmd
	return
}

func exists(path string, size int64) bool {
	s, err := os.Stat(path)
	if nil != err || s.Size() < size {
		return false
	}
	return true
}
