package main

import (
	"sync"
	"io"
	"time"
	"unicode/utf8"
	"strconv"
	"fmt"
)

type ReportType int
const (
	REPORT_IP   = ReportType(iota)
	REPORT_HOST
	REPORT_PATH
	REPORT_URL
	REPORT_FILE
)
func (rt ReportType) String() string {
	switch rt {
	case REPORT_IP:
		return "ip"
	case REPORT_HOST:
		return "host"
	case REPORT_PATH:
		return "path"
	case REPORT_URL:
		return "url"
	case REPORT_FILE:
		return "file"
	default:
		return ""
	}
}
func recommendReportCap(rt ReportType, length int) int {
	var r int
	switch rt {
	case REPORT_IP:
		r = 4096
	case REPORT_HOST:
		r = 1024
	case REPORT_PATH:
		r = 64 * 1024
	case REPORT_URL:
		r = 64 * 1024
	case REPORT_FILE:
		r = 64 * 1024
	default:
		return 0
	}
	length += length / 2
	if length > r {
		return length
	}
	return r
} 

type ReportData struct {
	Req     uint64
	DataReq uint64
	Body    uint64
}

type Report struct {
	m map[Name]*ReportData
	lock sync.RWMutex
}
func NewReport(rt ReportType, length int) *Report {
	_cap := recommendReportCap(rt, length)
	return &Report{
		m: make(map[Name]*ReportData, _cap),
	}
}
func (r *Report) Inc(name Name, data ReportData) {
	r.lock.Lock()
	v, ok := r.m[name]
	if !ok {
		v = new(ReportData)
		r.m[name] = v
	}
	v.Req += data.Req
	if v.Body != 0 {
		v.DataReq += data.Req
	}
	v.Body += data.Body
	r.lock.Unlock()
}
func (r *Report) Snapshot(rs *ReportSnapshot) *ReportSnapshot {
	if rs == nil {
		rs = new(ReportSnapshot)
	}
	r.lock.RLock()
	items := make([]nameDataPair, len(r.m))
	n := 0
	for name, data := range r.m {
		items[n].name = name
		items[n].data = *data
		n++
	}
	r.lock.RUnlock()
	rs.items = items
	return rs
}
func (r *Report) loadSnapshotData(rs *ReportSnapshot) {
	for i := range rs.items {
		r.m[rs.items[i].name] = &rs.items[i].data
	}
}

type nameDataPair struct {
	name Name
	data ReportData
}
type ReportSnapshot struct {
	report        *Report
	filename      string
	begin, end    time.Time
	items         []nameDataPair
}
func NewReportSnapshot(r *Report, filename string) *ReportSnapshot {
	return &ReportSnapshot{
		report: r,
		filename: filename,
	}
}
func (rs *ReportSnapshot) Filename() string {
	return rs.filename
}
func (rs *ReportSnapshot) SetTimespan(begin time.Time, duration time.Duration) {
	rs.begin    = begin
	rs.end      = begin.Add(duration)
}

var comma = []byte{ ',' }
var quote = []byte{ '"' }
var hex = "0123456789abcdef"
func (rs *ReportSnapshot) WriteJson(w io.Writer) error {
	if rs.items == nil {
		rs.report.Snapshot(rs)
	}

	var err error
	if _, err = fmt.Fprintf(w,
		`{"begin":"%s","end":"%s","archive":%t,"length":%d,"fields":["req","dreq","body"],"names":[`,
		rs.begin.Format(time.RFC3339), rs.end.Format(time.RFC3339), rs.filename != "",
		len(rs.items)); err != nil {
		return err
	}
	for i, item := range rs.items {
		if i > 0 {
			if _, err = w.Write(comma); err != nil {
				return err
			}
		}
		if err = encodeString(w, item.name.String()); err != nil {
			return err
		}
	}
	if _, err = w.Write([]byte(`],"data":[`)); err != nil {
		return err
	}
	var buffer [24]byte
	for i, item := range rs.items {
		if i > 0 {
			w.Write(comma)
		}
		w.Write(strconv.AppendUint(buffer[:0], item.data.Req, 10))
		w.Write(comma)
		w.Write(strconv.AppendUint(buffer[:0], item.data.DataReq, 10))
		w.Write(comma)
		w.Write(strconv.AppendUint(buffer[:0], item.data.Body, 10))	
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{ ']', '}' })
	return err
}

func encodeString(w io.Writer, s string) error {
	var err error	
	if _, err = w.Write(quote); err != nil {
		return err
	}
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				if _, err = w.Write([]byte(s[start:i])); err != nil {
					return err
				}
			}
			switch b {
				case '\\', '"':
					_, err = w.Write([]byte{ '\\', b })
				case '\n':
					_, err = w.Write([]byte{ '\\', 'n' })
				case '\r':
					_, err = w.Write([]byte{ '\\', 'r' })
				default:
					_, err = w.Write([]byte{ '\\', 'u', '0', '0', hex[b>>4], hex[b&0xF] })
			}
			if err != nil {
				return err
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				if _, err = w.Write([]byte(s[start:i])); err != nil {
					return err
				}
			}
			if _, err = w.Write([]byte{ '\\', 'u', 'f', 'f', 'f', 'd' }); err != nil {
				return err
			}
			i += size
			start = i
			continue
		}
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				if _, err = w.Write([]byte(s[start:i])); err != nil {
					return err
				}
			}
			if _, err = w.Write([]byte{ '\\', 'u', '2', '0', '2', hex[c&0xF] }); err != nil {
				return err
			}
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		if _, err = w.Write([]byte(s[start:])); err != nil {
			return err
		}
	}
	_, err = w.Write(quote)
	return err
}
