package main

import (
	log "github.com/Sirupsen/logrus"
	"os"
	"fmt"
	"strings"
)

type LoggingConfig struct {
	Filename string
	Format   string
	Level    string
}

const DEFAULT_FORMAT = "TEXT"

var OUTPUT_FILE *os.File

type PlainFormatter struct {

}

func (self *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	bytes := []byte(fmt.Sprintf("[%s %s] %s\n", entry.Time, strings.ToUpper(entry.Level.String()), entry.Message))
	return bytes, nil
}

func InitializeLogging(config *LoggingConfig, useStd bool, level string) {
	var lvl = log.DebugLevel
	var e error
	if level != "" {
		lvl, e = log.ParseLevel(level)
	} else if config.Level != "" {
		lvl, e = log.ParseLevel(config.Level)
	}
	if nil != e {
		lvl = log.DebugLevel
	}
	if useStd {
		log.SetOutput(os.Stdout)
		OUTPUT_FILE = nil
	} else {
		f, e := os.OpenFile(config.Filename, os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0666)
		if nil != e {
			fmt.Errorf("Open file <%s> for logging failed<%v>!\n", config.Filename, e)
		} else {
			log.SetOutput(f)
			OUTPUT_FILE = f
		}
	}
	if strings.ToLower(config.Format) == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&PlainFormatter{})
	}
	log.SetLevel(lvl)
	//log.Info("Logging Initialized.")
}

func DeinitializeLogging() {
	if nil != OUTPUT_FILE {
		OUTPUT_FILE.Close()
	}
}
