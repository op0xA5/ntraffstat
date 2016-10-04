package main

import (
	"io"
	"bufio"
	"bytes"
	"strings"
	"strconv"
	"net/url"
	"time"
)

/*
type parserOp struct {
	typ int
	r   rune
}
type RecordParser []parserOp
func (rp RecordParser) Parse(s string) *Record {
	var r [16]string
}*/

type RecordFlags int
const (
	StripFullUrl = RecordFlags(1 << iota)
	NoRefer
)

type RawRecord struct {
	Time  time.Time
	Ip    string
	Host  string
	Path  string
	Url   string
	File  string
	Body  uint64
	Refer string
}
type RecordReader struct {
	RawRecord
	bufio.Scanner

	flags RecordFlags
	valid bool
}
func NewRecordReader(r io.Reader) *RecordReader {
	return &RecordReader{
		Scanner: *bufio.NewScanner(r),
	}
}
func (rr *RecordReader) Scan() bool {
	rr.valid = false

	ok := rr.Scanner.Scan()
	if !ok {
		return false
	}

	rr.valid = rr.ParseFlags(rr.Bytes(), rr.flags)
	return true
}
func (rr *RecordReader) AddFlag(flags RecordFlags) {
	rr.flags |= flags
}
func (rr *RecordReader) Valid() bool {
	return rr.valid
}

func (rr *RawRecord) Parse(s []byte) bool {
	return rr.ParseFlags(s, 0)
}
func (rr *RawRecord) ParseFlags(s []byte, flags RecordFlags) bool {
	if len(s) == 0 {
		return false
	}

	rr.Time = time.Now()

	var p int
	p = bytes.IndexByte(s, ' ')
	if p == -1 {
		return false
	}
	ip := s[:p]
	s = bytes.TrimLeftFunc(s[p:], isAnsiSpace)

	if len(s) == 0 || s[0] != '"' {
		return false
	}
	s = s[1:]
	p = bytes.IndexByte(s, '"')
	if p == -1 {
		return false
	}
	req := s[:p]
	s = bytes.TrimLeftFunc(s[p+1:], isAnsiSpace)

	p = bytes.IndexByte(s, ' ')
	if p == -1 {
		return false
	}
	body, err := strconv.ParseUint(string(s[:p]), 10, 64)
	if err != nil {
		return false
	}
	s = bytes.TrimLeftFunc(s[p:], isAnsiSpace)

	if len(s) == 0 || s[0] != '"' {
		return false
	}
	s = s[1:]
	p = bytes.IndexByte(s, '"')
	if p == -1 {
		return false
	}
	file := s[:p]
	s = bytes.TrimLeftFunc(s[p+1:], isAnsiSpace)

	if len(s) == 0 || s[0] != '"' {
		return false
	}
	s = s[1:]
	p = bytes.IndexByte(s, '"')
	if p == -1 {
		return false
	}

	rr.Ip    = string(ip)
	rr.File  = string(file)
	rr.Body  = body

	if flags & NoRefer != 0 {
		if flags & StripFullUrl != 0 {
			rr.Host, rr.Path, _ = parseUrl(string(req))
			rr.Url = ""
		} else {
			rr.Host, rr.Path, rr.Url = parseUrl(string(req))
		}
		rr.Refer = ""	
	} else {
		refer := s[:p]
		if flags & StripFullUrl != 0 {
			rr.Host, rr.Path, _ = parseUrl(string(req))
			rr.Url = ""

			pos := bytes.IndexByte(refer, '?')
			if pos != -1 {
				refer = refer[:p]
			}
		} else {
			rr.Host, rr.Path, rr.Url = parseUrl(string(req))
		}

		rr.Refer, err = url.QueryUnescape(string(refer))
		if err != nil {
			rr.Refer = string(s[:p])
		}
	}

	return true
}
func isAnsiSpace(r rune) bool {
	return r == ' '
}
func parseUrl(s string) (host, path, full string) {
	hIdx := strings.IndexByte(s, '/')
	if hIdx == -1 {
		return s, s, s
	}
	if hIdx == len(s) - 1 {
		return s[:hIdx], s, s
	}
	path, err := url.QueryUnescape(s)
	if err != nil {
		path = s
	}
	pIdx := strings.IndexByte(path[hIdx:], '?')
	if pIdx == -1 {
		return s[:hIdx], path, path
	}
	return s[:hIdx], path[:hIdx+pIdx], path
}
