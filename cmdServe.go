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
	"os/exec"
	"path"
	"sync"

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
	format       string
	timeout      int
	concurrent   int
	downloadChan chan *downloadRequest
	cache        map[string]*downloadResult
	cacheMutex   sync.Mutex
)

type downloadRequest struct {
	id   string
	Name string `json:"name"`
	URL  string `json:"url"`
}

type downloadResult struct {
	Code     int    `json:"code"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   int    `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"msg"`
	_url     string
}

func writeResponse(w http.ResponseWriter, status int, data interface{}) {
	if bs, e := json.Marshal(data); nil != e {
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "plain/text")
		_, _ = w.Write([]byte(e.Error()))
	} else {
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bs)
	}
}

func md5sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func getDownload(id string) (*downloadResult, bool) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	r, b := cache[id]
	return r, b
}

func getAllDownloads() []*downloadResult {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	var results []*downloadResult
	for _, v := range cache {
		results = append(results, v)
	}
	return results
}

func setDownload(ret *downloadResult) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	cache[ret.ID] = ret
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if id != "" {
		if v, b := getDownload(id); b {
			writeResponse(w, 200, v)
		} else {
			writeResponse(w, 404, nil)
		}
	} else {
		writeResponse(w, 200, getAllDownloads())
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugln("downloadHandler:>")
	var req downloadRequest
	if bs, e := ioutil.ReadAll(r.Body); nil != e {
		http.Error(w, "invalid data:"+e.Error(), 400)
		return
	} else if e := json.Unmarshal(bs, &req); nil != e {
		http.Error(w, "invalid data:"+e.Error(), 400)
		return
	} else if req.Name == "" {
		http.Error(w, "invalid data: empty name", 400)
		return
	} else if req.URL == "" {
		http.Error(w, "invalid data: empty url", 400)
		return
	} else {
		log.Debugln("downloadHandler:>", req.Name, req.URL)
		req.id = md5sum(req.URL)
		if len(downloadChan) >= concurrent {
			var resp = downloadResult{
				Code:    -1,
				ID:      "",
				Name:    req.Name,
				Status:  -1,
				Message: "Tasks full, please wait...",
			}
			writeResponse(w, 200, resp)
		} else {
			downloadChan <- &req
			var resp = downloadResult{
				Code:    0,
				ID:      req.id,
				Name:    req.Name,
				Status:  0,
				Message: "",
			}
			writeResponse(w, 200, resp)
		}

	}
}

func downloadTask() {
	log.Infoln("downloadTask:> started")
	hlsgetter := NewHLSGetter(nil, root, nil, NewSegmentRewriter("%04d.ts"), 5, timeout, true, true, "", 5, 0)
	for t := range downloadChan {
		log.Debugln("downloadTask:>", t.id, t.Name, t.URL)
		go func() {
			log.Debugln(">>>> Downloading:", t.id, t.URL)
			var ret = downloadResult{
				ID:       t.id,
				Name:     t.Name,
				Status:   0,
				Progress: 0,
				Message:  "",
				_url:     t.URL,
			}
			// setDownload(&ret)
			// time.Sleep(30 * time.Second)
			ret.Status = 1
			setDownload(&ret)
			var dir string
			var index string
			var mp4 string
			if combine {
				dir = path.Join(root, ".tmp", t.id)
				mp4 = path.Join(root, t.Name+"."+format)
			} else {
				dir = path.Join(root, t.Name)
			}
			index = path.Join(dir, "index.m3u8")
			hlsgetter.Download(t.URL, dir, "index.m3u8", func(url string, dest string, ret_code int, ret_msg string) {
				if ret_code != 0 {
					ret.Status = -1
					ret.Message = ret_msg
				} else {
					ret.Status = 2
				}
				setDownload(&ret)
			})
			if combine && mp4 != "" && ffmpeg != "" {
				cmd := exec.Command(ffmpeg, "-i", index, "-c", "copy", mp4)
				if e := cmd.Run(); nil != e {
					log.Errorln("Run command failed:", e)
					ret.Status = -1
					ret.Message = e.Error()
				} else {
					ret.Status = 3
				}
				setDownload(&ret)
			}
			log.Debugln(">>>> Downloaded:", t.id, t.URL)
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
		router.HandleFunc("/status/", statusHandler)
		router.HandleFunc("/status/{id}", statusHandler)
		router.PathPrefix("/").Handler(
			handlers.CombinedLoggingHandler(os.Stdout, http.StripPrefix("/", http.FileServer(htmldocs.Assets))))
		log.Infoln("Start to serve @", listen, "...")
		if concurrent < 1 {
			concurrent = 5
		}
		downloadChan = make(chan *downloadRequest, concurrent)
		cache = make(map[string]*downloadResult)
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
	serveCmd.Flags().BoolVar(&combine, "combine", false, "Combine segments into MP4/TS file")
	serveCmd.Flags().StringVar(&format, "format", "ts", "Combine file format")
	serveCmd.Flags().IntVar(&timeout, "timeout", 20, "Request timeout in seconds.")
	serveCmd.Flags().IntVar(&concurrent, "concurrent", 5, "Concurrent download tasks")
}
