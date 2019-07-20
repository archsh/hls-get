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
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hls-get",
	Short: "A magic program running as a service.",
	Long:  `For more information, please contact author.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
	Run: downloadFunc,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVar(&downloadCfg.Output, "O", ".", "Output directory.")
	//PR 'path_rewrite'    - [STRING] Rewrite output path method. Default empty means simple copy.
	rootCmd.Flags().StringVar(&downloadCfg.PathRewrite, "PR", "", "Rewrite output path method. Empty means simple copy.")
	//SR 'segment_rewrite'     - [STRING] Rewrite segment name method. Default empty means simple copy.
	rootCmd.Flags().StringVar(&downloadCfg.SegmentRewrite, "SR", "", "Rewrite segment name method. Empty means simple copy.")
	//UA 'user_agent'    - [STRING] UserAgent. Default is 'hls-get' with version num.
	rootCmd.Flags().StringVar(&downloadCfg.UserAgent, "UA", "hls-get "+VERSION+"("+TAG+")", "UserAgent.")
	//L  'log'   - [STRING] Logging output file. Default 'stdout'.
	rootCmd.Flags().StringVar(&downloadCfg.LogFile, "L", "", "Logging output file. Default 'stdout'.")
	//V 'loglevel' - [STRING] Log level. Default 'INFO'.
	rootCmd.Flags().StringVar(&downloadCfg.LogLevel, "V", "INFO", "Logging level. Default 'INFO'.")
	//R  'retry' - [INTEGER] Retry times if download fails.
	rootCmd.Flags().IntVar(&downloadCfg.Retries, "R", 0, "Retry times if download fails.")
	//S  'skip'  - [BOOL] Skip if exists.
	rootCmd.Flags().BoolVar(&downloadCfg.Skip, "S", false, "Skip if exists.")
	//SZ 'skip_on_size' - [BOOL] Skip if size different.
	rootCmd.Flags().BoolVar(&downloadCfg.SkipOnSize, "SZ", false, "Skip if size different.")
	//M  'mode'  - [STRING] Source mode: redis, mysql. Default empty means source via command args.
	rootCmd.Flags().StringVar(&downloadCfg.Mode, "M", "", "Source mode: redis, mysql. Empty means source via command args.")
	//RD 'redirect'   - [STRING] Redirect server request.
	rootCmd.Flags().StringVar(&downloadCfg.Redirect, "RR", "", "Redirect server request.")
	//C  'concurrent' - [INTEGER] Concurrent tasks.
	rootCmd.Flags().IntVar(&downloadCfg.Concurrent, "CO", 5, "Concurrent tasks.")
	//TO 'timeout'    - [INTEGER] Request timeout in seconds.
	rootCmd.Flags().IntVar(&downloadCfg.Timeout, "TO", 20, "Request timeout in seconds.")
	//TT 'total'      - [INTEGER] Total download links.
	rootCmd.Flags().Int64Var(&downloadCfg.Total, "TT", 0, "Total download links.")
	///// Redis Configurations =========================================================================================
	//RH 'redis_host'  - [STRING] Redis host.
	rootCmd.Flags().StringVar(&downloadCfg.Redis.Host, "RH", "localhost", "Redis host.")
	//RP 'redis_port'  - [INTEGER] Redis port.
	rootCmd.Flags().IntVar(&downloadCfg.Redis.Port, "RP", 6379, "Redis port.")
	//RD 'redis_db'    - [INTEGER] Redis db num.
	rootCmd.Flags().IntVar(&downloadCfg.Redis.Db, "RD", 0, "Redis db num.")
	//RW 'redis_password'  - [STRING] Redis password.
	rootCmd.Flags().StringVar(&downloadCfg.Redis.Password, "RW", "", "Redis password.")
	//RK 'redis_key'   - [STRING] List key name in redis.
	rootCmd.Flags().StringVar(&downloadCfg.Redis.Key, "RK", "HLSGET_DOWNLOADS", "List key name in redis.")
	///// MySQL Configurations =========================================================================================
	//MH 'mysql_host'  - [STRING] MySQL host.
	rootCmd.Flags().StringVar(&downloadCfg.MySQL.Host, "MH", "localhost", "MySQL host.")
	//MP 'mysql_port'  - [INTEGER] MySQL port.
	rootCmd.Flags().IntVar(&downloadCfg.MySQL.Port, "MP", 3306, "MySQL port.")
	//MN 'mysql_username' - [STRING] MySQL username.
	rootCmd.Flags().StringVar(&downloadCfg.MySQL.Username, "MN", "root", "MySQL username.")
	//MW 'mysql_password' - [STRING] MySQL password.
	rootCmd.Flags().StringVar(&downloadCfg.MySQL.Password, "MW", "", "MySQL password.")
	//MD 'mysql_db'       - [STRING] MySQL database.
	rootCmd.Flags().StringVar(&downloadCfg.MySQL.Db, "MD", "hlsgetdb", "MySQL database.")
	//MT 'mysql_table'    - [STRING] MySQL table.
	rootCmd.Flags().StringVar(&downloadCfg.MySQL.Table, "MT", "hlsget_downloads", "MySQL table.")
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config",
		"/etc/hls-get.yaml", "config file (default is /etc/hls-get.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.AddCommand(downloadCmd, serveCmd, checkCmd, schemaCmd, versionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".hls-get" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".hls-get")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})
	Execute()
}
