package main

import (
	"log"
	"net"
	. "net/http"
	"mime"
	"path"
	"os"
	"io"
	"bufio"
	"sync"
	"time"
	"strings"
	"compress/gzip"
)

const FILE_ALLOW_IPS = "allow_ip"
const FILE_USERS     = "allow_users"
const PATH_STATIC    = "static"
const INDEX_FILE     = "index.html"

const GZIP_LEVEL     = gzip.DefaultCompression

var allowIpsNextCheck  time.Time
var allowIpsLastModify time.Time
var allowIpsCache      []*net.IPNet
var allowIpsMutex      sync.Mutex
func readAllowIps() []*net.IPNet {
	allowIpsMutex.Lock()
	defer allowIpsMutex.Unlock()
	now := time.Now()
	if now.Before(allowIpsNextCheck) {
		return allowIpsCache
	}

	fi, err := os.Stat(FILE_ALLOW_IPS)
	if os.IsNotExist(err) {
		allowIpsNextCheck  = now.Add(100 * time.Millisecond)
		allowIpsLastModify = time.Time{}
		allowIpsCache      = nil
		return allowIpsCache
	}
	if err != nil {
		log.Printf("error read file: %s  %v\n", FILE_ALLOW_IPS, err)
		return allowIpsCache
	}
	if fi.ModTime() == allowIpsLastModify {
		return allowIpsCache
	}

	f, err := os.Open(FILE_ALLOW_IPS)
	defer f.Close()
	allowIpsNextCheck  = now.Add(100 * time.Millisecond)
	allowIpsLastModify = fi.ModTime()
	allowIpsCache      = make([]*net.IPNet, 0, 32)
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum ++
		line := strings.TrimSpace(sc.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		_, v, err := net.ParseCIDR(line)
		if err != nil {
			ip := net.ParseIP(line)
			if ip == nil {
				log.Printf("invalid format: %s #%d\n", FILE_ALLOW_IPS, lineNum)
				continue
			}
			v = &net.IPNet{ ip, net.CIDRMask(len(ip)*8, len(ip)*8) }
		}
		allowIpsCache = append(allowIpsCache, v)
	}
	if err := sc.Err(); err != nil {
		log.Printf("error read file: %s  %v\n", FILE_ALLOW_IPS, err)
	}
	log.Printf("allow ip loaded\n")
	return allowIpsCache
}
var usersNextCheck  time.Time
var usersLastModify time.Time
var usersCache      map[string]bool
var usersMutex      sync.Mutex
func readUsers() map[string]bool {
	usersMutex.Lock()
	defer usersMutex.Unlock()
	now := time.Now()
	if now.Before(usersNextCheck) {
		return usersCache
	}

	fi, err := os.Stat(FILE_USERS)
	if os.IsNotExist(err) {
		usersNextCheck  = now.Add(100 * time.Millisecond)
		usersLastModify = time.Time{}
		usersCache      = nil
		return usersCache
	}
	if err != nil {
		return usersCache
	}
	if fi.ModTime() == usersLastModify {
		return usersCache
	}

	f, err := os.Open(FILE_USERS)
	defer f.Close()
	usersNextCheck  = now.Add(100 * time.Millisecond)
	usersLastModify = fi.ModTime()
	usersCache      = make(map[string]bool, 32)
	sc := bufio.NewScanner(f)
	lineNum := 0
	for sc.Scan() {
		lineNum ++
		line := strings.TrimSpace(sc.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		if strings.IndexByte(line, '=') == -1 {
			log.Printf("invalid format: %s #%d\n", FILE_USERS, lineNum)
			continue
		}
		usersCache[line] = true
	}
	if err := sc.Err(); err != nil {
		log.Printf("error read file: %s  %v\n", FILE_USERS, err)
	}
	log.Printf("login users loaded\n")
	return usersCache
}

type HttpMux struct {
	ServeMux
}
func NewHttpMux() *HttpMux {
	return &HttpMux{
		*NewServeMux(),
	}
}
func (hm *HttpMux) ServeHTTP(w ResponseWriter, r *Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		Error(w, "400 Bad Request", StatusBadRequest)
		return
	}
	ip := net.ParseIP(host)
	if ip == nil {
		Error(w, "400 Bad Request", StatusBadRequest)
		return
	}
	matched := false
	if ip.IsLoopback() {
		matched = true
	} else {
		for _, m := range readAllowIps() {
			if m.Contains(ip) {
				matched = true
				break
			}
		}
	}
	if !matched {
		Error(w, "403 Forbidden", StatusForbidden)
		return
	}

	um := readUsers()
	if um != nil {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"Login Required\"")
			Error(w, "401 Unauthorized", StatusUnauthorized)
			return
		}
		if !um[user+"="+pass] {
			Error(w, "401 Unauthorized", StatusUnauthorized)
			return
		}
	}

	hm.ServeMux.ServeHTTP(w, r)
}
func getBodyWriter(w ResponseWriter, r *Request) io.WriteCloser {
	if GZIP_LEVEL != 0 && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		if gzip_w, err := gzip.NewWriterLevel(w, GZIP_LEVEL); err == nil {
			h := w.Header()
			h.Set("Content-Encoding", "gzip")
			delete(h, "Content-Length")
			w.WriteHeader(StatusOK)
			return gzip_w
		}
	}
	w.WriteHeader(StatusOK)
	return nopWriteCloser{ w }
}
type nopWriteCloser struct {
	io.Writer
}
func (nwc nopWriteCloser) Close() error { return nil }

type staticHandler struct{}
func (_ staticHandler) ServeHTTP(w ResponseWriter, r *Request) {
	file := path.Join(PATH_STATIC, r.URL.Path)
	fi, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("file not exists: %s\n", file)
			Error(w, "404 Not Found", StatusNotFound)
			return
		}
		log.Printf("error open file: %s %v\n", file, err)
		Error(w, "403 Forbidden", StatusForbidden)
		return
	}
	if fi.IsDir() {
		if r.URL.Path[len(r.URL.Path)-1] != '/' {
			Redirect(w, r, path.Base(r.URL.Path) + "/", StatusMovedPermanently)
			return
		}
		file = path.Join(file, INDEX_FILE)
		fi, err = os.Stat(file)
		if err != nil {
			Error(w, "403 Forbidden", StatusForbidden)
			return
		}
	}

	h := w.Header()
	modtime := fi.ModTime()
	if t, err := time.Parse(TimeFormat, r.Header.Get("If-Modified-Since"));
		err == nil && modtime.Before(t.Add(1*time.Second)) {
		delete(h, "Content-Type")
		delete(h, "Content-Length")
		w.WriteHeader(StatusNotModified)
		return
	}

	f, err := os.Open(file)
	if err != nil {
		log.Printf("error open file: %s %v\n", file, err)
		Error(w, "500 Internal Server Error", StatusInternalServerError)
		return
	}
	defer f.Close()

	h.Set("Content-Type", mime.TypeByExtension(path.Ext(fi.Name())))
	h.Set("Last-Modified", modtime.UTC().Format(TimeFormat))
	bw := getBodyWriter(w, r)
	defer bw.Close()
	w.WriteHeader(StatusOK)
	io.Copy(bw, f)
}

type tableHandler struct{}
func (_ tableHandler) ServeHTTP(w ResponseWriter, r *Request) {
	var table  *Table
	var report ReportType
	var ti     time.Time
	p := strings.Split(r.URL.Path, "/")
	if len(p) != 4 {
		Error(w, "404 Not Found", StatusNotFound)
		return
	}
	switch p[1] {
	case "m5":
		table = tbl_m5
	case "h":
		table = tbl_h
	case "d":
		table = tbl_d
	default:
		Error(w, "404 Not Found", StatusNotFound)
		return
	}
	switch p[2] {
	case "ip":
		report = REPORT_IP
	case "host":
		report = REPORT_HOST
	case "path":
		report = REPORT_PATH
	case "url":
		report = REPORT_URL
	case "file":
		report = REPORT_FILE
	default:
		Error(w, "404 Not Found", StatusNotFound)
		return
	}
	realtime := p[3] == "realtime"
	if !realtime {
		switch len(p[3]) {
		case len("200601021504"):
			ti, _ = time.ParseInLocation("200601021504", p[3], time.Local)
		case len("2006010215"):
			ti, _ = time.ParseInLocation("2006010215", p[3], time.Local)
		case len("20060102"):
			ti, _ = time.ParseInLocation("20060102", p[3], time.Local)
		}
		if ti.IsZero() {
			Error(w, "404 Not Found", StatusNotFound)
			return
		}
	}

	if realtime || table.IsCurrent(ti) {
		snap := table.GetCurrentSnapshot(report)
		if snap == nil {
			Error(w, "404 Not Found", StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/json")
		bw := getBodyWriter(w, r)
		defer bw.Close()
		snap.WriteJson(bw)
	} else {
		file := table.GetSavedFilename(report, ti)
		f, err := os.Open(path.Join("logs", file))
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("file not exists: %s\n", file)
				Error(w, "404 Not Found", StatusNotFound)
				return
			}
			log.Printf("error open file: %s %v\n", file, err)
			Error(w, "403 Forbidden", StatusForbidden)
			return
		}
		defer f.Close()

		h := w.Header()
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.Set("Content-Encoding", "gzip")
			h.Set("Content-Type", "text/json")
			delete(h, "Content-Length")
			w.WriteHeader(StatusOK)
			io.Copy(w, f)
			return
		}
		h.Set("Content-Type", "text/json")
		delete(h, "Content-Length")
		gr, err := gzip.NewReader(f)
		if err != nil {
			Error(w, "500 Internal Server Error", StatusInternalServerError)
			return
		}
		w.WriteHeader(StatusOK)
		io.Copy(w, gr)
	}
}

type trendHandler struct{}
func (_ trendHandler) ServeHTTP(w ResponseWriter, r *Request) {
	var tr *Trend
	var ti time.Time
	p := strings.Split(r.URL.Path, "/")
	if len(p) != 3 {
		Error(w, "404 Not Found", StatusNotFound)
		return
	}
	switch p[1] {
	case "m":
		tr = tr_m
	case "h":
		tr = tr_h
	default:
		Error(w, "404 Not Found", StatusNotFound)
		return
	}	
	realtime := p[2] == "realtime"
	if !realtime {
		switch len(p[2]) {
		case len("2006010215"):
			ti, _ = time.ParseInLocation("2006010215", p[2], time.Local)
		case len("20060102"):
			ti, _ = time.ParseInLocation("20060102", p[2], time.Local)
		}
		if ti.IsZero() {
			Error(w, "404 Not Found", StatusNotFound)
			return
		}
	}

	var file string
	if realtime {
		file = tr.GetCurrentFilename()
	} else {
		file = tr.GetSavedFilename(ti)
	}
	f, err := os.Open(path.Join("logs", file))
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("file not exists: %s\n", file)
			Error(w, "404 Not Found", StatusNotFound)
			return
		}
		log.Printf("error open file: %s %v\n", file, err)
		Error(w, "403 Forbidden", StatusForbidden)
		return
	}
	defer f.Close()

	h := w.Header()
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		h.Set("Content-Encoding", "gzip")
		h.Set("Content-Type", "text/json")
		delete(h, "Content-Length")
		w.WriteHeader(StatusOK)
		io.Copy(w, f)
		return
	}
	h.Set("Content-Type", "text/json")
	delete(h, "Content-Length")
	gr, err := gzip.NewReader(f)
	if err != nil {
		Error(w, "500 Internal Server Error", StatusInternalServerError)
		return
	}
	w.WriteHeader(StatusOK)
	io.Copy(w, gr)	
}

type referHandler struct{}
func (_ referHandler) ServeHTTP(w ResponseWriter, r *Request) {
	var rt *RefererTable
	var np *NamePool
	switch r.URL.Path {
	case "/url_refer":
		rt = ref_url_refer
		np = np_url
	case "/path_refer":
		rt = ref_path_refer
		np = np_url
	case "/file_path":
		rt = ref_file_path
		np = np_file
	default:
		Error(w, "404 Not Found", StatusNotFound)
		return
	}
	q := r.FormValue("q")
	if q == "" {
		Error(w, "[]", StatusOK)
		return
	}
	name := np.Get(q)
	if name.IsNil() {
		Error(w, "[]", StatusOK)
		return
	}
	res := rt.Find(name)
	if len(res) == 0 {
		Error(w, "[]", StatusOK)
		return
	}
	w.Header().Set("Content-Type", "text/json")
	bw := getBodyWriter(w, r)
	defer bw.Close()
	res.WriteJson(bw)
}

func startHttpServer(l net.Listener) {
	mux := NewHttpMux()

	mux.Handle("/table/", StripPrefix("/table", tableHandler{}))
	mux.Handle("/trend/", StripPrefix("/trend", trendHandler{}))
	mux.Handle("/refer/", StripPrefix("/refer", referHandler{}))
	mux.Handle("/urlinfo", urlInfoHandler{})
	mux.Handle("/fileinfo", fileInfoHandler{})
	mux.Handle("/", staticHandler{})

	Serve(l, mux)
}
