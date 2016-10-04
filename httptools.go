package main

import (
	. "net/http"
	"fmt"
	"os"
	"path"
	"time"
)

var UrlInfoAllowedHeader = []string{
	"Cache-Control",
	"Content-Language",
	"Content-Length",
	"Content-Location",
	"Content-Type",
	"Date",
	"ETag",
	"Expires",
	"Last-Modified",
	"Location",
	"Pragma",
	"Server",
	"Vary",
	"Via",
}

func stripProtocolScheme(uri string) (string, bool) {
	for i, r := range uri {
		if r == ':' {
			if len(uri) < i + 3 ||
				uri[i+1] != '/' || 
				uri[i+2] != '/' {
				return uri, false
			}
			return uri[i+3:], true
		}
		if !(('a' <= r && r <= 'z') ||
			('A' <= r && r <= 'Z')) {
			return uri, false
		}
	}
	return uri, false
}

type urlInfoHandler struct{}
func (_ urlInfoHandler) ServeHTTP(w ResponseWriter, r *Request) {
	q := r.FormValue("q")
	if q == "" {
		Error(w, "[]", StatusOK)
		return
	}

	if _, ok := stripProtocolScheme(q); !ok {
		q = "http://"+q
	}

	req, _ := NewRequest("HEAD", q, nil)
	if req == nil {
		Error(w, "500 Internal Server Error", StatusInternalServerError)
		return
	}
	client := new(Client)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprint(w, `["Status: -1",`)
		encodeString(w, "Error: "+err.Error())
		w.Write([]byte{ ']' })
		return
	}

	bw := getBodyWriter(w, r)
	defer bw.Close()
	fmt.Fprintf(bw, `["Status: %s","Proto: %s"`, resp.Status, resp.Proto)
	for _, h := range UrlInfoAllowedHeader {
		if vs, has := resp.Header[h]; has {
			for _, v := range vs {
				bw.Write([]byte{ ',' })
				encodeString(bw, h+": "+v)
			}
		}
	}
	bw.Write([]byte{ ']' })
}

type fileInfoHandler struct{}
func (_ fileInfoHandler) ServeHTTP(w ResponseWriter, r *Request) {
	q := r.FormValue("q")
	if q == "" {
		Error(w, "[]", StatusOK)
		return
	}

	fi, err := os.Stat(path.Join(Config.FileRoot, q))
	if err != nil {
		w.Write([]byte{ '[' })
		encodeString(w, "Error: "+err.Error())
		w.Write([]byte{ ']' })
		return
	}

	bw := getBodyWriter(w, r)
	defer bw.Close()
	fmt.Fprintf(bw, `["Size: %d","Mode: %s","ModTime: %s"]`,
		fi.Size(), fi.Mode().String(), fi.ModTime().Format(time.RFC1123))
}

