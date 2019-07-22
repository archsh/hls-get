package main

import (
	"flag"
	"github.com/spf13/pflag"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
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

var (
	downloadCfg Configuration
)

func downloadFunc(cmd *cobra.Command, args []string) {
	if downloadCfg.UserAgent != "" {
		downloadCfg.UserAgent = "hls-get " + VERSION + "(" + TAG + ")"
	}
	if downloadCfg.Retries < 1 {
		downloadCfg.Retries = 1
	}
	if downloadCfg.Concurrent < 1 {
		downloadCfg.Concurrent = 5
	}
	if downloadCfg.SkipOnSize {
		downloadCfg.Skip = downloadCfg.SkipOnSize
	}

	if logging_config.Filename != "" {
		InitializeLogging(&logging_config, false, logging_config.Level)
	} else {
		InitializeLogging(&logging_config, true, logging_config.Level)
	}
	defer DeinitializeLogging()
	pathRewriter := NewPathRewriter(downloadCfg.PathRewrite)
	segmentRewriter := NewSegmentRewriter(downloadCfg.SegmentRewrite)
	var dlInterface DL_Interface
	var loop bool
	if downloadCfg.Mode == "mysql" {
		// Fetch list from MySQL.
		log.Infoln("Using mysql as task dispatcher...")
		dlInterface = NewMySQLDl(downloadCfg.MySQL.Host, uint(downloadCfg.MySQL.Port), downloadCfg.MySQL.Db, downloadCfg.MySQL.Table, downloadCfg.MySQL.Username, downloadCfg.MySQL.Password)
		loop = true
	} else if downloadCfg.Mode == "redis" {
		// Fetch list from Redis.
		log.Infoln("Using redis as task dispatcher...")
		dlInterface = NewRedisDl(downloadCfg.Redis.Host, uint(downloadCfg.Redis.Port), downloadCfg.Redis.Password, downloadCfg.Redis.Db, downloadCfg.Redis.Key)
		loop = true
	} else if len(args) > 0 {
		// Fetch list from Args.
		log.Infoln("Using download list from arguments ...")
		dlInterface = NewDummyDl(args)
	} else {
		Usage()
		_, _ = os.Stderr.Write([]byte("\n"))
		return
	}
	hlsgetter := NewHLSGetter(dlInterface, downloadCfg.Output, pathRewriter, segmentRewriter, downloadCfg.Retries,
		downloadCfg.Timeout, downloadCfg.Skip, downloadCfg.SkipOnSize, downloadCfg.Redirect, downloadCfg.Concurrent, downloadCfg.Total)
	hlsgetter.SetUA(downloadCfg.UserAgent)
	hlsgetter.Run(loop)
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download via Command line",
	Run:   downloadFunc,
}
var flagSet  pflag.FlagSet

func init() {
	flagSet.StringVar(&downloadCfg.Output, "output", ".", "Output directory.")
	//PR 'path_rewrite'    - [STRING] Rewrite output path method. Default empty means simple copy.
	flagSet.StringVar(&downloadCfg.PathRewrite, "path_rewrite", "", "Rewrite output path method. Empty means simple copy.")
	//SR 'segment_rewrite'     - [STRING] Rewrite segment name method. Default empty means simple copy.
	flagSet.StringVar(&downloadCfg.SegmentRewrite, "segment_rewrite", "", "Rewrite segment name method. Empty means simple copy.")
	//UA 'user_agent'    - [STRING] UserAgent. Default is 'hls-get' with version num.
	flagSet.StringVar(&downloadCfg.UserAgent, "user_agent", "hls-get "+VERSION+"("+TAG+")", "UserAgent.")
	//L  'log'   - [STRING] Logging output file. Default 'stdout'.
	flagSet.StringVar(&downloadCfg.LogFile, "log_file", "", "Logging output file. Default 'stdout'.")
	//V 'loglevel' - [STRING] Log level. Default 'INFO'.
	flagSet.StringVar(&downloadCfg.LogLevel, "log_level", "INFO", "Logging level. Default 'INFO'.")
	//R  'retry' - [INTEGER] Retry times if download fails.
	flagSet.IntVar(&downloadCfg.Retries, "retries", 0, "Retry times if download fails.")
	//S  'skip'  - [BOOL] Skip if exists.
	flagSet.BoolVar(&downloadCfg.Skip, "skip_exists", false, "Skip if exists.")
	//SZ 'skip_on_size' - [BOOL] Skip if size different.
	flagSet.BoolVar(&downloadCfg.SkipOnSize, "skip_on_size", false, "Skip if size different.")
	//M  'mode'  - [STRING] Source mode: redis, mysql. Default empty means source via command args.
	flagSet.StringVar(&downloadCfg.Mode, "mode", "", "Source mode: redis, mysql. Empty means source via command args.")
	//RD 'redirect'   - [STRING] Redirect server request.
	flagSet.StringVar(&downloadCfg.Redirect, "redirect", "", "Redirect server request.")
	//C  'concurrent' - [INTEGER] Concurrent tasks.
	flagSet.IntVar(&downloadCfg.Concurrent, "concurrent", 5, "Concurrent tasks.")
	//TO 'timeout'    - [INTEGER] Request timeout in seconds.
	flagSet.IntVar(&downloadCfg.Timeout, "timeout", 20, "Request timeout in seconds.")
	//TT 'total'      - [INTEGER] Total download links.
	flagSet.Int64Var(&downloadCfg.Total, "total", 0, "Total download links.")
	///// Redis Configurations =========================================================================================
	//RH 'redis_host'  - [STRING] Redis host.
	flagSet.StringVar(&downloadCfg.Redis.Host, "redis_host", "localhost", "Redis host.")
	//RP 'redis_port'  - [INTEGER] Redis port.
	flagSet.IntVar(&downloadCfg.Redis.Port, "redis_port", 6379, "Redis port.")
	//RD 'redis_db'    - [INTEGER] Redis db num.
	flagSet.IntVar(&downloadCfg.Redis.Db, "redis_db", 0, "Redis db num.")
	//RW 'redis_password'  - [STRING] Redis password.
	flagSet.StringVar(&downloadCfg.Redis.Password, "redis_password", "", "Redis password.")
	//RK 'redis_key'   - [STRING] List key name in redis.
	flagSet.StringVar(&downloadCfg.Redis.Key, "redis_key", "HLSGET_DOWNLOADS", "List key name in redis.")
	///// MySQL Configurations =========================================================================================
	//MH 'mysql_host'  - [STRING] MySQL host.
	flagSet.StringVar(&downloadCfg.MySQL.Host, "mysql_host", "localhost", "MySQL host.")
	//MP 'mysql_port'  - [INTEGER] MySQL port.
	flagSet.IntVar(&downloadCfg.MySQL.Port, "mysql_port", 3306, "MySQL port.")
	//MN 'mysql_username' - [STRING] MySQL username.
	flagSet.StringVar(&downloadCfg.MySQL.Username, "mysql_user", "root", "MySQL username.")
	//MW 'mysql_password' - [STRING] MySQL password.
	flagSet.StringVar(&downloadCfg.MySQL.Password, "mysql_password", "", "MySQL password.")
	//MD 'mysql_db'       - [STRING] MySQL database.
	flagSet.StringVar(&downloadCfg.MySQL.Db, "mysql_db", "hlsgetdb", "MySQL database.")
	//MT 'mysql_table'    - [STRING] MySQL table.
	flagSet.StringVar(&downloadCfg.MySQL.Table, "mysql_table", "hlsget_downloads", "MySQL table.")

	downloadCmd.Flags().AddFlagSet(&flagSet)
}
