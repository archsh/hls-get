package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hls-get/htmldocs"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

var (
	debug        bool
	listen       string
	root         string
	combine      bool
	ffmpeg       string
	timeout      int
	concurrent   int
	downloadChan chan *downloadReq
)

type downloadReq struct {
	id   string
	Name string `json:"name"`
	Url  string `json:"url"`
}

type downloadResp struct {
	Code    int    `json:"code"`
	Id      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"msg"`
}

func writeResponse(w http.ResponseWriter, code int, id string, status int, message string) {
	var resp = downloadResp{
		Code:    code,
		Id:      id,
		Status:  status,
		Message: message,
	}
	if bs, e := json.Marshal(resp); nil != e {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "plain/text")
		_, _ = w.Write([]byte("OK"))
	} else {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bs)
	}
}

func md5sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
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
		req.id = md5sum(req.Url)
		if len(downloadChan) >= concurrent {
			writeResponse(w, -1, "", -1, "Tasks full, please wait...")
		} else {
			downloadChan <- &req
			writeResponse(w, 0, req.id, 0, "")
		}

	}
}

func downloadTask() {
	log.Infoln("downloadTask:> started")
	for t := range downloadChan {
		log.Debugln("downloadTask:>", t.id, t.Name, t.Url)
		go func() {
			log.Debugln(">>>> Downloading:", t.id, t.Url)
			time.Sleep(30 * time.Second)
			log.Debugln(">>>> Downloaded:", t.id, t.Url)
		}()
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
		if concurrent < 1 {
			concurrent = 5
		}
		downloadChan = make(chan *downloadReq, concurrent)
		go downloadTask()
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
	serveCmd.Flags().IntVar(&concurrent, "concurrent", 5, "Concurrent download tasks")
}
