package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"hls-get/htmldocs"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	debug   bool
	listen  string
	root    string
	combine bool
	ffmpeg  string
	timeout int
)

type downloadReq struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugln("downloadHandler:>")
	var req downloadReq
	if bs, e := ioutil.ReadAll(r.Body); nil != e {
		http.Error(w, "invalid data:"+e.Error(), 400)
		return
	} else if e := json.Unmarshal(bs, &req); nil != e {
		http.Error(w, "invalid data:"+e.Error(), 400)
		return
	} else if req.Name == "" {
		http.Error(w, "invalid data: empty name", 400)
		return
	} else if req.Url == "" {
		http.Error(w, "invalid data: empty url", 400)
		return
	} else {
		log.Debugln("downloadHandler:>", req.Name, req.Url)
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "plain/text")
		_, _ = w.Write([]byte("OK"))
	}
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve as a HTTP WEB Application",
	Run: func(cmd *cobra.Command, args []string) {
		InitializeLogging(&logging_config, debug, "DEBUG")
		defer DeinitializeLogging()

		var router = mux.NewRouter().StrictSlash(true)
		//router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//	http.Redirect(w, r, "/admin/", 302)
		//})
		router.HandleFunc("/download", downloadHandler)
		router.PathPrefix("/").Handler(
			handlers.CombinedLoggingHandler(os.Stdout, http.StripPrefix("/", http.FileServer(htmldocs.Assets))))
		log.Infoln("Start to serve @", listen, "...")
		if e := http.ListenAndServe(listen, handlers.CombinedLoggingHandler(os.Stdout, router)); nil != e {
			fmt.Println("Serve failed:>", e)
		}
	},
}

func init() {
	serveCmd.Flags().BoolVar(&debug, "debug", false, "Debug mod")
	serveCmd.Flags().StringVar(&listen, "listen", ":8080", "HTTP Listen Address")
	serveCmd.Flags().StringVar(&root, "root", "", "Root directory to save files")
	serveCmd.Flags().StringVar(&ffmpeg, "ffmpeg", "ffmpeg", "FFMPEG executable path")
	serveCmd.Flags().BoolVar(&combine, "combine", false, "Combine segments into MP4 file")
	serveCmd.Flags().IntVar(&timeout, "timeout", 20, "Request timeout in seconds.")
}
