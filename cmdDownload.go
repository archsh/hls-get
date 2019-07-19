package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
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

var (
	downloadCfg Configuration
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download via Command line",
	Run: func(cmd *cobra.Command, args []string) {
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
	},
}

func init() {
	downloadCmd.Flags().StringVar(&downloadCfg.Output, "O", ".", "Output directory.")
	//PR 'path_rewrite'    - [STRING] Rewrite output path method. Default empty means simple copy.
	downloadCmd.Flags().StringVar(&downloadCfg.PathRewrite, "PR", "", "Rewrite output path method. Empty means simple copy.")
	//SR 'segment_rewrite'     - [STRING] Rewrite segment name method. Default empty means simple copy.
	downloadCmd.Flags().StringVar(&downloadCfg.SegmentRewrite, "SR", "", "Rewrite segment name method. Empty means simple copy.")
	//UA 'user_agent'    - [STRING] UserAgent. Default is 'hls-get' with version num.
	downloadCmd.Flags().StringVar(&downloadCfg.UserAgent, "UA", "hls-get "+VERSION+"("+TAG+")", "UserAgent.")
	//L  'log'   - [STRING] Logging output file. Default 'stdout'.
	downloadCmd.Flags().StringVar(&downloadCfg.LogFile, "L", "", "Logging output file. Default 'stdout'.")
	//V 'loglevel' - [STRING] Log level. Default 'INFO'.
	downloadCmd.Flags().StringVar(&downloadCfg.LogLevel, "V", "INFO", "Logging level. Default 'INFO'.")
	//R  'retry' - [INTEGER] Retry times if download fails.
	downloadCmd.Flags().IntVar(&downloadCfg.Retries, "R", 0, "Retry times if download fails.")
	//S  'skip'  - [BOOL] Skip if exists.
	downloadCmd.Flags().BoolVar(&downloadCfg.Skip, "S", false, "Skip if exists.")
	//SZ 'skip_on_size' - [BOOL] Skip if size different.
	downloadCmd.Flags().BoolVar(&downloadCfg.SkipOnSize, "SZ", false, "Skip if size different.")
	//M  'mode'  - [STRING] Source mode: redis, mysql. Default empty means source via command args.
	downloadCmd.Flags().StringVar(&downloadCfg.Mode, "M", "", "Source mode: redis, mysql. Empty means source via command args.")
	//RD 'redirect'   - [STRING] Redirect server request.
	downloadCmd.Flags().StringVar(&downloadCfg.Redirect, "RR", "", "Redirect server request.")
	//C  'concurrent' - [INTEGER] Concurrent tasks.
	downloadCmd.Flags().IntVar(&downloadCfg.Concurrent, "CO", 5, "Concurrent tasks.")
	//TO 'timeout'    - [INTEGER] Request timeout in seconds.
	downloadCmd.Flags().IntVar(&downloadCfg.Timeout, "TO", 20, "Request timeout in seconds.")
	//TT 'total'      - [INTEGER] Total download links.
	downloadCmd.Flags().Int64Var(&downloadCfg.Total, "TT", 0, "Total download links.")
	///// Redis Configurations =========================================================================================
	//RH 'redis_host'  - [STRING] Redis host.
	downloadCmd.Flags().StringVar(&downloadCfg.Redis.Host, "RH", "localhost", "Redis host.")
	//RP 'redis_port'  - [INTEGER] Redis port.
	downloadCmd.Flags().IntVar(&downloadCfg.Redis.Port, "RP", 6379, "Redis port.")
	//RD 'redis_db'    - [INTEGER] Redis db num.
	downloadCmd.Flags().IntVar(&downloadCfg.Redis.Db, "RD", 0, "Redis db num.")
	//RW 'redis_password'  - [STRING] Redis password.
	downloadCmd.Flags().StringVar(&downloadCfg.Redis.Password, "RW", "", "Redis password.")
	//RK 'redis_key'   - [STRING] List key name in redis.
	downloadCmd.Flags().StringVar(&downloadCfg.Redis.Key, "RK", "HLSGET_DOWNLOADS", "List key name in redis.")
	///// MySQL Configurations =========================================================================================
	//MH 'mysql_host'  - [STRING] MySQL host.
	downloadCmd.Flags().StringVar(&downloadCfg.MySQL.Host, "MH", "localhost", "MySQL host.")
	//MP 'mysql_port'  - [INTEGER] MySQL port.
	downloadCmd.Flags().IntVar(&downloadCfg.MySQL.Port, "MP", 3306, "MySQL port.")
	//MN 'mysql_username' - [STRING] MySQL username.
	downloadCmd.Flags().StringVar(&downloadCfg.MySQL.Username, "MN", "root", "MySQL username.")
	//MW 'mysql_password' - [STRING] MySQL password.
	downloadCmd.Flags().StringVar(&downloadCfg.MySQL.Password, "MW", "", "MySQL password.")
	//MD 'mysql_db'       - [STRING] MySQL database.
	downloadCmd.Flags().StringVar(&downloadCfg.MySQL.Db, "MD", "hlsgetdb", "MySQL database.")
	//MT 'mysql_table'    - [STRING] MySQL table.
	downloadCmd.Flags().StringVar(&downloadCfg.MySQL.Table, "MT", "hlsget_downloads", "MySQL table.")
}
