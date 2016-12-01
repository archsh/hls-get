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

import (
	"flag"
	"os"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/archsh/hlsutils/helpers/logging"
	"github.com/BurntSushi/toml"
)

const VERSION = "0.9.15"

var logging_config = logging.LoggingConfig{Format:logging.DEFAULT_FORMAT, Level:"INFO"}

type RedisConfig struct {
	Host string
	Port int
	Db int
	Password string
	Key string
}

type MySQLConfig struct {
	Host string
	Port int
	Username string
	Password string
	Db string
	Table string
}

type Configuration struct {
	Output          string
	Path_Rewrite    string
	Segment_Rewrite string
	User_Agent      string
	Log_File        string
	Log_Level       string
	Retries         int
	Skip            bool
	Skip_On_Size    bool
	Mode            string
	Redirect        string
	Concurrent      int
	Timeout         int
	Total           int64
	Redis           RedisConfig
	MySQL           MySQLConfig
}

func Usage() {
	guide := `
Scenarios:
  (1) Simple mode: download one or multiple URL without DB support.
  (2) Redis support: download multiple URL via REDIS LIST.
  (3) MySQL support: download multiple URL via MySQL DB Table.

Usage:
  hls-get [OPTIONS,...] [URL1,URL2,...]

Options:
`
	os.Stdout.Write([]byte(guide))
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
	var mysql_show_schema bool
	flag.BoolVar(&mysql_show_schema, "MS", false, "Only show MySQL table schema. Will not do any furthur action if this is set.")
	var checkConfig bool
	flag.BoolVar(&checkConfig, "C", false, "Check configration only.")
	///// Global Options ===============================================================================================
	//O  'output'     - [STRING] Output directory. Default '.'.
	flag.StringVar(&cfg.Output, "O", ".", "Output directory.")
	//PR 'path_rewrite'    - [STRING] Rewrite output path method. Default empty means simple copy.
	flag.StringVar(&cfg.Path_Rewrite, "PR", "", "Rewrite output path method. Empty means simple copy.")
	//SR 'segment_rewrite'     - [STRING] Rewrite segment name method. Default empty means simple copy.
	flag.StringVar(&cfg.Segment_Rewrite, "SR", "", "Rewrite segment name method. Empty means simple copy.")
	//UA 'user_agent'    - [STRING] UserAgent. Default is 'hls-get' with version num.
	flag.StringVar(&cfg.User_Agent, "UA", "hls-get v" + VERSION, "UserAgent.")
	//L  'log'   - [STRING] Logging output file. Default 'stdout'.
	flag.StringVar(&cfg.Log_File, "L", "", "Logging output file. Default 'stdout'.")
	//V 'loglevel' - [STRING] Log level. Default 'INFO'.
	flag.StringVar(&cfg.Log_Level, "V", "INFO", "Logging level. Default 'INFO'.")
	//R  'retry' - [INTEGER] Retry times if download fails.
	flag.IntVar(&cfg.Retries, "R", 0, "Retry times if download fails.")
	//S  'skip'  - [BOOL] Skip if exists.
	flag.BoolVar(&cfg.Skip, "S", false, "Skip if exists.")
	//SZ 'skip_on_size' - [BOOL] Skip if size different.
	flag.BoolVar(&cfg.Skip_On_Size, "SZ", false, "Skip if size different.")
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
		os.Stderr.Write([]byte(fmt.Sprintf("hls-get v%v\n", VERSION)))
		os.Exit(0)
	}

	os.Stderr.Write([]byte(fmt.Sprintf("hls-get v%v - HTTP Live Streaming (HLS) Downloader.\n", VERSION)))
	os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh@gmail.com>. Licensed for use under the GNU GPL version 3.\n"))
	if mysql_show_schema {
		ShowMySQLSchema()
		os.Exit(0)
	}
	if config != "" {
		cfg := new(Configuration)
		if _, e := toml.DecodeFile(config, cfg); nil != e {
			os.Stderr.Write([]byte(fmt.Sprintf("Load config<%s> failed: %s.\n", config, e)))
			os.Exit(1)
		}else{
			os.Stderr.Write([]byte(fmt.Sprintf("Loaded config from <%s> .\n", config)))
		}
	}
	if cfg.User_Agent != "" {
		cfg.User_Agent = "hls-get v" + VERSION
	}
	if cfg.Retries < 1 {
		cfg.Retries = 1
	}
	if cfg.Concurrent < 1 {
		cfg.Concurrent = 5
	}
	if cfg.Skip_On_Size {
		cfg.Skip = cfg.Skip_On_Size
	}

	if checkConfig {
		os.Stderr.Write([]byte("Current Config: \n\n"))
		toml.NewEncoder(os.Stderr).Encode(cfg)
		os.Exit(0)
	}

	if logging_config.Filename != "" {
		logging.InitializeLogging(&logging_config, false, logging_config.Level)
	}else{
		logging.InitializeLogging(&logging_config, true, logging_config.Level)
	}
	defer logging.DeinitializeLogging()
	path_rewriter := NewPathRewriter(cfg.Path_Rewrite)
	segment_rewriter := NewSegmentRewriter(cfg.Segment_Rewrite)
	var dl_interface DL_Interface
	var loop bool
	if cfg.Mode == "mysql" {
		// Fetch list from MySQL.
		log.Infoln("Using mysql as task dispatcher...")
		dl_interface = NewMySQLDl(cfg.MySQL.Host, uint(cfg.MySQL.Port), cfg.MySQL.Db, cfg.MySQL.Table, cfg.MySQL.Username, cfg.MySQL.Password)
		loop = true
	}else if cfg.Mode == "redis" {
		// Fetch list from Redis.
		log.Infoln("Using redis as task dispatcher...")
		dl_interface = NewRedisDl(cfg.Redis.Host, uint(cfg.Redis.Port), cfg.Redis.Password, cfg.Redis.Db, cfg.Redis.Key)
		loop = true
	}else if flag.NArg() > 0 {
		// Fetch list from Args.
		log.Infoln("Using download list from arguments ...")
		dl_interface = NewDummyDl(flag.Args())
	}else{
		Usage()
		os.Stderr.Write([]byte("\n"))
		return
	}
	hlsgetter := NewHLSGetter(dl_interface, cfg.Output, path_rewriter, segment_rewriter, cfg.Retries,
		cfg.Timeout, cfg.Skip, cfg.Skip_On_Size, cfg.Redirect, cfg.Concurrent, cfg.Total)
	hlsgetter.SetUA(cfg.User_Agent)
	hlsgetter.Run(loop)
}