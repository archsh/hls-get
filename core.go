package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/archsh/go.m3u8"
	//"path"
	//"regexp"
	"strings"
	"sync"
	"time"
)

type Download struct {
	URI           string
	Filename      string
	refer         string
	totalSegments uint
	index         uint
	retries       int
}

type HLSGetter struct {
	client          *http.Client
	dlInterface     DL_Interface
	pathRewriter    StringRewriter
	segmentRewriter StringRewriter

	output      string
	retries     int
	timeout     int
	skipExists  bool
	skipOnSize  bool
	userAgent   string
	concurrent  int
	redirectUrl string
	total       int64
}

func NewHLSGetter(dlInterface DL_Interface, output string,
	pathRewriter StringRewriter, segmentRewriter StringRewriter,
	retries int, timeout int, skipExists bool, skipOnSize bool, redirect string, concurrent int, total int64) *HLSGetter {
	hls := new(HLSGetter)
	hls.client = &http.Client{Timeout: time.Duration(timeout) * time.Second}
	hls.dlInterface = dlInterface
	hls.output = output
	hls.pathRewriter = pathRewriter
	hls.segmentRewriter = segmentRewriter
	hls.redirectUrl = redirect
	hls.retries = retries
	hls.timeout = timeout
	hls.skipExists = skipExists
	hls.skipOnSize = skipOnSize
	hls.concurrent = concurrent
	hls.userAgent = "hls-get v" + VERSION
	hls.total = total
	return hls
}

func (getter *HLSGetter) SetUA(ua string) {
	getter.userAgent = ua
}

func (getter *HLSGetter) PathRewrite(intput string) string {
	if getter.pathRewriter != nil {
		return getter.pathRewriter.RunString(intput, 0)
	}
	return intput
}

func (getter *HLSGetter) SegmentRewrite(input string, idx int) string {
	if getter.segmentRewriter != nil {
		return getter.segmentRewriter.RunString(input, idx)
	}
	return input
}

func (getter *HLSGetter) Run(loop bool) {
	if getter.concurrent < 1 {
		log.Fatalln("Concurrent can not less than 1!")
	}
	if getter.dlInterface == nil {
		log.Fatalln("Download List Interface can not be nil!")
	}
	var totalDownloaded int64
	var totalSuccess int64
	var totalFailed int64
	totalDownloaded = 0
	totalFailed = 0
	for {
		if getter.total > 0 && totalDownloaded >= getter.total {
			log.Infoln("Reached total of downloads:", getter.total)
			break;
		}
		var num int
		if getter.total > 0 && getter.concurrent > int(getter.total-totalDownloaded) {
			num = int(getter.total - totalDownloaded)
		} else {
			num = getter.concurrent
		}
		urls, err := getter.dlInterface.NextLinks(num)
		//log.Debugln("length of urls:", len(urls))
		if nil != err || len(urls) == 0 {
			log.Errorln("Can not get links!", err)
			if loop {
				time.Sleep(time.Second * 5)
				continue
			} else {
				break;
			}
		}
		var wg sync.WaitGroup
		wg.Add(len(urls))
		for _, l := range urls {
			log.Debugln(" Downloading ", l, "...")
			go func(lk string) {
				url, dest, retCode, retMsg := getter.Download(lk, getter.output, "", func(total, current, avails int, uri string) {
					log.Infof("Download: %d/%d/%d %s ...\b", current, avails, total, uri)
				})
				if retCode != 0 {
					totalFailed += 1
				} else {
					totalSuccess += 1
				}
				totalDownloaded += 1
				getter.dlInterface.SubmitResult(url, dest, retCode, retMsg)
				wg.Done()
			}(l)
		}
		wg.Wait()
		if len(urls) < getter.concurrent || len(urls) < 1 {
			log.Infoln("End of download list.")
			break
		}

	}
	log.Infoln("Total Downloaded:", totalDownloaded)
	log.Infoln("Total Success:", totalSuccess)
	log.Infoln("Total Failed:", totalFailed)
}

func (getter *HLSGetter) doRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", getter.userAgent)
	req.Header.Add("Accept-Encoding", "identity")
	resp, err := getter.client.Do(req)
	return resp, err
}

func (getter *HLSGetter) GetSegment(url string, filename string, skip_exists bool, skip_on_size bool, retries int) (string, error) {
	if skip_exists && exists(filename, 100) {
		if skip_on_size {
			if req, err := http.NewRequest("HEAD", url, nil); nil == err {
				if resp, err := getter.doRequest(req); nil == err && resp.StatusCode == 200 && exists(filename, resp.ContentLength) {
					log.Infof("Segment file '%s' exists with size %d bytes.\n", filename, resp.ContentLength)
					return filename, nil
				}
			}
		} else {
			log.Infof("Segment file '%s' exists.\n", filename)
			return filename, nil
		}
	}
	if retries < 1 {
		retries = 1
	}
	for i := 0; i < retries; i++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Errorf("GetSegment:1> Create new request failed: %v\n", err)
			return filename, err
		}
		resp, err := getter.doRequest(req)
		if err != nil {
			log.Errorf("GetSegment:4> do request failed: %v\n", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			log.Errorf("Received HTTP %v for %v \n", resp.StatusCode, url)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Errorf("GetSegment:5> Copy response body failed: %v\n", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		if "" != filename {
			err = os.MkdirAll(filepath.Dir(filename), 0777)
			if err != nil {
				log.Errorf("GetSegment:2> Create path %s failed :%v\n", filepath.Dir(filename), err)
				return filename, err
			}
		} else {
			//out, err = ioutil.TempFile("./", "__savedTempSegment")
			return filename, errors.New("Filename empty!!!")
		}
		out, err := os.Create(filename)
		defer out.Close()
		if err != nil {
			log.Errorf("Create file '%s' failed: %v\n", filename, err)
			return filename, err
		}
		if n, err := out.Write(respBody); nil != err {
			log.Errorf("Write segment file '%s' failed:> %s \n", filename, err)
			return filename, err
		} else {
			log.Infof("Write segment file '%s' %d bytes.", filename, n)
		}
		return filename, nil
	}
	return "", errors.New("failed to download segment")
}

func (getter *HLSGetter) GetPlaylist(urlStr string, outDir string, filename string, retries int, skip_exists bool) (segments []*Download, dest string, ret_code int, ret_msg string) {
	if retries < 1 {
		retries = 1
	}
	for i := 0; i < retries; i++ {
		if getter.redirectUrl != "" {
			urlStr = fmt.Sprintf(getter.redirectUrl, urlStr)
		}
		log.Debugln("GetPlaylist:> Get ", urlStr)
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			log.Errorf("GetPlaylist:> Failed to build request: %v \n", err)
			continue
		}
		resp, err := getter.doRequest(req)
		if err != nil {
			log.Errorln("GetPlaylist:> Request failed:", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		if filename == "" {
			filename = getter.PathRewrite(resp.Request.URL.Path)
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("GetPlaylist:> Read response failed: %v \n", err)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		buffer := bytes.NewBuffer(respBody)
		playlistFilename := filepath.Join(outDir, filename)
		err = os.MkdirAll(filepath.Dir(playlistFilename), 0777)
		if err != nil {
			log.Errorf("GetPlaylist:> Create directory '%s' failed: %v \n", filepath.Dir(playlistFilename), err)
			ret_code = -1
			ret_msg = fmt.Sprintf("Create directory '%s' failed: %v \n", filepath.Dir(playlistFilename), err)
			return
		}
		playlist, listType, err := m3u8.Decode(*buffer, true)
		if err != nil {
			log.Errorf("GetPlaylist:> Decode M3U8 failed: %v \n", err)
			continue
		}
		resp.Body.Close()
		if listType == m3u8.MEDIA {
			mpl := playlist.(*m3u8.MediaPlaylist)
			segs := len(mpl.Segments)
			new_mpl, err := m3u8.NewMediaPlaylist(uint(segs), uint(segs))
			for idx, v := range mpl.Segments {
				if v == nil {
					continue
				}
				var msURI string
				var msFilename string
				var segname string
				if strings.HasPrefix(v.URI, "http://") || strings.HasPrefix(v.URI, "https://") {
					msURI = v.URI
					segname = fmt.Sprintf("%04d.ts", idx)
				} else {
					msUrl, _ := resp.Request.URL.Parse(v.URI)
					msURI = msUrl.String()
					segname = v.URI
				}
				segname = getter.SegmentRewrite(v.URI, idx) //fmt.Sprintf("%04d.ts", idx)
				msFilename = filepath.Join(filepath.Dir(playlistFilename), segname)
				//mpl.Segments[idx].URI = segname
				newseg := m3u8.MediaSegment{
					SeqId:           v.SeqId,
					Title:           v.Title,
					URI:             segname,
					Duration:        v.Duration,
					Limit:           v.Limit,
					Offset:          v.Offset,
					Key:             v.Key,
					Map:             v.Map,
					Discontinuity:   v.Discontinuity,
					SCTE:            v.SCTE,
					ProgramDateTime: v.ProgramDateTime,
				}
				//mpl.Segments[idx].URI = segname
				//new_mpl.Append(segname, v.Duration, v.Title)
				new_mpl.AppendSegment(&newseg)
				segments = append(segments, &Download{msURI, msFilename, urlStr, uint(segs), uint(idx + 1), 0})
			}
			log.Debugln("GetPlaylist> Writing playlist to ", playlistFilename, "...")
			out, err := os.Create(playlistFilename)
			if err != nil {
				log.Errorf("GetPlaylist:10> %v \n", err)
				ret_code = -3
				ret_msg = fmt.Sprint(err)
				return
			}
			defer out.Close()
			new_mpl.Close()
			buf := new_mpl.Encode()
			if n, e := io.Copy(out, buf); nil != e {
				log.Errorf("Write playlist '%s' failed: %s \n", playlistFilename, e)
				return nil, "", -1, "Failed to write playlist."
			} else {
				log.Infof("Write playlist '%s' %d bytes.\n", playlistFilename, n)
			}
			//dest = playlistFilename
			return segments, playlistFilename, 0, ""
		} else {
			log.Errorln("GetPlaylist:11> Not a valid media playlist")
			time.Sleep(time.Duration(1) * time.Second)
		}
	}
	return nil, "", -1, "Failed to get playlist."
}

// callback(total,current,avails,segUri)
func (getter *HLSGetter) Download(urlStr string, outDir string, filename string, callback func(int, int, int, string)) (string, string, int, string) {
	var dest string
	var retCode int
	var retMsg string
	var segments []*Download
	failures := 0
	var total, current, avails int
	log.Debugln("Download> ", urlStr, outDir)
	segments, dest, retCode, retMsg = getter.GetPlaylist(urlStr, outDir, filename, getter.retries, getter.skipExists)
	total = len(segments)
	if total < 1 || retCode != 0 {
		//callback(urlStr, dest, retCode, retMsg)
		callback(total, current, avails, "")
		return urlStr, dest, retCode, retMsg
	}

	for i := 0; i < total; i += getter.concurrent {
		var segs []*Download
		if i+getter.concurrent < total {
			segs = segments[i : i+getter.concurrent]
		} else {
			segs = segments[i:]
		}
		var wg sync.WaitGroup
		wg.Add(len(segs))
		for j, seg := range segs {
			//log.Debugln(">>> Seg:", seg.URI)
			go func(ps *Download) {
				s, e := getter.GetSegment(ps.URI, ps.Filename, getter.skipExists, getter.skipOnSize, getter.retries)
				if e != nil {
					failures += 1
					log.Errorln("Download Segment failed:", ps.URI, e)
				} else {
					log.Debugln("Downloaded >", s)
					current = i + j
					avails += 1
					callback(total, current, avails, "")
				}
				wg.Done()
			}(seg)
		}
		wg.Wait()
	}

	if failures > 1 {
		return urlStr, dest, -9, fmt.Sprintf("Failed to download %d segments!", failures)
	} else {
		return urlStr, dest, 0, ""
	}
}
