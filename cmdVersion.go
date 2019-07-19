package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version",
	Run: func(cmd *cobra.Command, args []string) {
		//_, _ = os.Stderr.Write([]byte(fmt.Sprintf("hls-get %v (%s) - HTTP Live Streaming (HLS) Downloader.\n", VERSION, TAG)))
		_, _ = os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh@gmail.com>. Licensed for use under the GNU GPL version 3.\n"))
		_, _ = os.Stderr.Write([]byte(fmt.Sprintf("hls-get %v (%s), Built @ %s \n", VERSION, TAG, BUILD_TIME)))
	},
}
