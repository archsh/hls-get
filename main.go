/*
   hls-get

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package main

//go:generate sh ./gen_version.sh version.go
import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"os"
)

var logging_config = LoggingConfig{Format: DEFAULT_FORMAT, Level: "INFO"}

type RedisConfig struct {
	Host     string
	Port     int
	Db       int
	Password string
	Key      string
}

type MySQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Db       string
	Table    string
}

type Configuration struct {
	Output         string
	PathRewrite    string
	SegmentRewrite string
	UserAgent      string
	LogFile        string
	LogLevel       string
	Retries        int
	Skip           bool
	SkipOnSize     bool
	Mode           string
	Redirect       string
	Concurrent     int
	Timeout        int
	Total          int64
	Redis          RedisConfig
	MySQL          MySQLConfig
}

func Usage() {
	guide := `
Scenarios:
  (1) Simple mode: download one or multiple URL without Database support.
  (2) Redis support: download multiple URL via REDIS LIST.
  (3) MySQL support: download multiple URL via MySQL Database Table.

Usage:
  hls-get [OPTIONS,...] [URL1,URL2,...]

Options:
`
	_, _ = os.Stdout.Write([]byte(guide))
	flag.PrintDefaults()
}

/***********************************************************************************************************************
 * MAIN ENTRY
 *
 */
func main() {
	cfg := new(Configuration)

	///// Control Options ==============================================================================================
	//c 'config'    - [STRING] Config file instead of the following parameters. Default empty.
	var config string
	flag.StringVar(&config, "c", "", "Use a config file instead of the other parameters. Default empty.")
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "Display version info.")
	var mysqlShowSchema bool
	flag.BoolVar(&mysqlShowSchema, "MS", false, "Only show MySQL table schema. Will not do any furthur action if this is set.")
	var checkConfig bool
	flag.BoolVar(&checkConfig, "C", false, "Check configration only.")
	///// Global Options ===============================================================================================
	//O  'output'     - [STRING] Output directory. Default '.'.
	flag.StringVar(&cfg.Output, "O", ".", "Output directory.")
	//PR 'path_rewrite'    - [STRING] Rewrite output path method. Default empty means simple copy.
	flag.StringVar(&cfg.PathRewrite, "PR", "", "Rewrite output path method. Empty means simple copy.")
	//SR 'segment_rewrite'     - [STRING] Rewrite segment name method. Default empty means simple copy.
	flag.StringVar(&cfg.SegmentRewrite, "SR", "", "Rewrite segment name method. Empty means simple copy.")
	//UA 'user_agent'    - [STRING] UserAgent. Default is 'hls-get' with version num.
	flag.StringVar(&cfg.UserAgent, "UA", "hls-get "+VERSION+"("+TAG+")", "UserAgent.")
	//L  'log'   - [STRING] Logging output file. Default 'stdout'.
	flag.StringVar(&cfg.LogFile, "L", "", "Logging output file. Default 'stdout'.")
	//V 'loglevel' - [STRING] Log level. Default 'INFO'.
	flag.StringVar(&cfg.LogLevel, "V", "INFO", "Logging level. Default 'INFO'.")
	//R  'retry' - [INTEGER] Retry times if download fails.
	flag.IntVar(&cfg.Retries, "R", 0, "Retry times if download fails.")
	//S  'skip'  - [BOOL] Skip if exists.
	flag.BoolVar(&cfg.Skip, "S", false, "Skip if exists.")
	//SZ 'skip_on_size' - [BOOL] Skip if size different.
	flag.BoolVar(&cfg.SkipOnSize, "SZ", false, "Skip if size different.")
	//M  'mode'  - [STRING] Source mode: redis, mysql. Default empty means source via command args.
	flag.StringVar(&cfg.Mode, "M", "", "Source mode: redis, mysql. Empty means source via command args.")
	//RD 'redirect'   - [STRING] Redirect server request.
	flag.StringVar(&cfg.Redirect, "RR", "", "Redirect server request.")
	//C  'concurrent' - [INTEGER] Concurrent tasks.
	flag.IntVar(&cfg.Concurrent, "CO", 5, "Concurrent tasks.")
	//TO 'timeout'    - [INTEGER] Request timeout in seconds.
	flag.IntVar(&cfg.Timeout, "TO", 20, "Request timeout in seconds.")
	//TT 'total'      - [INTEGER] Total download links.
	flag.Int64Var(&cfg.Total, "TT", 0, "Total download links.")
	///// Redis Configurations =========================================================================================
	//RH 'redis_host'  - [STRING] Redis host.
	flag.StringVar(&cfg.Redis.Host, "RH", "localhost", "Redis host.")
	//RP 'redis_port'  - [INTEGER] Redis port.
	flag.IntVar(&cfg.Redis.Port, "RP", 6379, "Redis port.")
	//RD 'redis_db'    - [INTEGER] Redis db num.
	flag.IntVar(&cfg.Redis.Db, "RD", 0, "Redis db num.")
	//RW 'redis_password'  - [STRING] Redis password.
	flag.StringVar(&cfg.Redis.Password, "RW", "", "Redis password.")
	//RK 'redis_key'   - [STRING] List key name in redis.
	flag.StringVar(&cfg.Redis.Key, "RK", "HLSGET_DOWNLOADS", "List key name in redis.")
	///// MySQL Configurations =========================================================================================
	//MH 'mysql_host'  - [STRING] MySQL host.
	flag.StringVar(&cfg.MySQL.Host, "MH", "localhost", "MySQL host.")
	//MP 'mysql_port'  - [INTEGER] MySQL port.
	flag.IntVar(&cfg.MySQL.Port, "MP", 3306, "MySQL port.")
	//MN 'mysql_username' - [STRING] MySQL username.
	flag.StringVar(&cfg.MySQL.Username, "MN", "root", "MySQL username.")
	//MW 'mysql_password' - [STRING] MySQL password.
	flag.StringVar(&cfg.MySQL.Password, "MW", "", "MySQL password.")
	//MD 'mysql_db'       - [STRING] MySQL database.
	flag.StringVar(&cfg.MySQL.Db, "MD", "hlsgetdb", "MySQL database.")
	//MT 'mysql_table'    - [STRING] MySQL table.
	flag.StringVar(&cfg.MySQL.Table, "MT", "hlsget_downloads", "MySQL table.")

	flag.Parse()

	if showVersion {
		_, _ = os.Stderr.Write([]byte(fmt.Sprintf("hls-get %v (%s), Built @ %s \n", VERSION, TAG, BUILD_TIME)))
		os.Exit(0)
	}

	_, _ = os.Stderr.Write([]byte(fmt.Sprintf("hls-get %v (%s) - HTTP Live Streaming (HLS) Downloader.\n", VERSION, TAG)))
	_, _ = os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh@gmail.com>. Licensed for use under the GNU GPL version 3.\n"))
	if mysqlShowSchema {
		ShowMySQLSchema()
		os.Exit(0)
	}
	if config != "" {
		cfg := new(Configuration)
		if _, e := toml.DecodeFile(config, cfg); nil != e {
			_, _ = os.Stderr.Write([]byte(fmt.Sprintf("Load config<%s> failed: %s.\n", config, e)))
			os.Exit(1)
		} else {
			_, _ = os.Stderr.Write([]byte(fmt.Sprintf("Loaded config from <%s> .\n", config)))
		}
	}
	if cfg.UserAgent != "" {
		cfg.UserAgent = "hls-get " + VERSION + "(" + TAG + ")"
	}
	if cfg.Retries < 1 {
		cfg.Retries = 1
	}
	if cfg.Concurrent < 1 {
		cfg.Concurrent = 5
	}
	if cfg.SkipOnSize {
		cfg.Skip = cfg.SkipOnSize
	}

	if checkConfig {
		_, _ = os.Stderr.Write([]byte("Current Config: \n\n"))
		_ = toml.NewEncoder(os.Stderr).Encode(cfg)
		os.Exit(0)
	}

	if logging_config.Filename != "" {
		InitializeLogging(&logging_config, false, logging_config.Level)
	} else {
		InitializeLogging(&logging_config, true, logging_config.Level)
	}
	defer DeinitializeLogging()
	pathRewriter := NewPathRewriter(cfg.PathRewrite)
	segmentRewriter := NewSegmentRewriter(cfg.SegmentRewrite)
	var dlInterface DL_Interface
	var loop bool
	if cfg.Mode == "mysql" {
		// Fetch list from MySQL.
		log.Infoln("Using mysql as task dispatcher...")
		dlInterface = NewMySQLDl(cfg.MySQL.Host, uint(cfg.MySQL.Port), cfg.MySQL.Db, cfg.MySQL.Table, cfg.MySQL.Username, cfg.MySQL.Password)
		loop = true
	} else if cfg.Mode == "redis" {
		// Fetch list from Redis.
		log.Infoln("Using redis as task dispatcher...")
		dlInterface = NewRedisDl(cfg.Redis.Host, uint(cfg.Redis.Port), cfg.Redis.Password, cfg.Redis.Db, cfg.Redis.Key)
		loop = true
	} else if flag.NArg() > 0 {
		// Fetch list from Args.
		log.Infoln("Using download list from arguments ...")
		dlInterface = NewDummyDl(flag.Args())
	} else {
		Usage()
		_, _ = os.Stderr.Write([]byte("\n"))
		return
	}
	hlsgetter := NewHLSGetter(dlInterface, cfg.Output, pathRewriter, segmentRewriter, cfg.Retries,
		cfg.Timeout, cfg.Skip, cfg.SkipOnSize, cfg.Redirect, cfg.Concurrent, cfg.Total)
	hlsgetter.SetUA(cfg.UserAgent)
	hlsgetter.Run(loop)
}
