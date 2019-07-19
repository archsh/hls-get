package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"hls-get/htmldocs"
	"net/http"
	"os"
)

var (
	debug      bool
	listenAddr = ":8080"
)
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve as a HTTP WEB Application",
	Run: func(cmd *cobra.Command, args []string) {
		InitializeLogging(&logging_config, debug, "DEBUG")
		defer DeinitializeLogging()

		var router = mux.NewRouter().StrictSlash(true)
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", 302)
		})
		router.PathPrefix("/admin/").Handler(
			handlers.CombinedLoggingHandler(os.Stdout, http.StripPrefix("/admin/", http.FileServer(htmldocs.Assets))))
		logrus.Infoln("Start to serve @", listenAddr, "...")
		if e := http.ListenAndServe(listenAddr, handlers.CombinedLoggingHandler(os.Stdout, router)); nil != e {
			fmt.Println("Serve failed:>", e)
		}
	},
}

func init() {
	serveCmd.Flags().BoolVar(&debug, "debug", false, "Debug mod")
	serveCmd.Flags().StringVar(&listenAddr, "listen", ":8080", "HTTP Listen Address")
}
