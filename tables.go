package main

import (
	"time"
	"strings"
)

type Record struct {
	Time    time.Time

	Ip      Name
	Host    Name
	Path    Name
	Url     Name
	File    Name
	Body    uint64
	Refer   Name
}

type Table struct {
	reportGroup

	update    Truncater
	last      time.Time 
	next      time.Time
	begin     time.Time

	savefile string  // eg: `2016/0102/m5-1504-{report}.json.gz`
	w        *DumpWorker
}

type reportGroup struct {
	ip   *Report
	host *Report
	path *Report
	url  *Report
	file *Report
}
func newReportGroup() reportGroup {
	return reportGroup{
		ip:   NewReport(REPORT_IP, 0),
		host: NewReport(REPORT_HOST, 0),
		path: NewReport(REPORT_PATH, 0),
		url:  NewReport(REPORT_URL, 0),
		file: NewReport(REPORT_FILE, 0),
	}
}
func NewTable(begin time.Time, update Truncater) *Table {
	last := update.Truncate(begin)
	next := last.Add(update.Duration(begin))

	return &Table{
		reportGroup: newReportGroup(),

		update:   update,
		last:     last,
		next:     next,
		begin:    begin,
	}
}
func (t *Table) SetSavefile(filename string, w *DumpWorker) {
	t.savefile = filename
	t.w = w
}
func (t *Table) Add(r Record) {
	for !r.Time.Before(t.next) {
		t._flush()
	}

	rd := ReportData{ Req: 1, Body: r.Body }
	t.ip.Inc(r.Ip, rd)
	t.host.Inc(r.Host, rd)
	t.path.Inc(r.Path, rd)
	t.url.Inc(r.Url, rd)
	t.file.Inc(r.File, rd)
}
func (t *Table) Flush(time time.Time) {
	for !time.Before(t.next) {
		t._flush()
	}
}
func (t *Table) _flush() {
	last, duration := t.last, t.update.Duration(t.next)
	t.last, t.next = t.next, t.next.Add(duration)

	old := t.reportGroup
	t.reportGroup = newReportGroup()

	if t.w != nil {
		begin := last
		if begin.Before(t.begin) {
			begin = t.begin
			duration -= begin.Sub(last)
		}
		ipSnap := NewReportSnapshot(old.ip, t.GetSavedFilename(REPORT_IP, last))
		ipSnap.SetTimespan(begin, duration)
		t.w.Add(ipSnap)
		hostSnap := NewReportSnapshot(old.host, t.GetSavedFilename(REPORT_HOST, last))
		hostSnap.SetTimespan(begin, duration)
		t.w.Add(hostSnap)
		pathSnap := NewReportSnapshot(old.path, t.GetSavedFilename(REPORT_PATH, last))
		pathSnap.SetTimespan(begin, duration)
		t.w.Add(pathSnap)
		urlSnap := NewReportSnapshot(old.url, t.GetSavedFilename(REPORT_URL, last))
		urlSnap.SetTimespan(begin, duration)
		t.w.Add(urlSnap)
		fileSnap := NewReportSnapshot(old.file, t.GetSavedFilename(REPORT_FILE, last))
		fileSnap.SetTimespan(begin, duration)
		t.w.Add(fileSnap)
	}
}
func (t *Table) GetCurrentSnapshot(rt ReportType) *ReportSnapshot {
	var r *Report
	switch rt {
	case REPORT_IP:
		r = t.ip
	case REPORT_HOST:
		r = t.host
	case REPORT_PATH:
		r = t.path
	case REPORT_URL:
		r = t.url
	case REPORT_FILE:
		r = t.file
	default:
		return nil
	}
	snap := r.Snapshot(nil)
	begin := t.last
	if begin.Before(t.begin) {
		begin = t.begin
	}
	snap.SetTimespan(begin, time.Now().Sub(begin))
	return snap
}
func (t *Table) IsCurrent(ti time.Time) bool {
	return !t.last.After(ti) && t.next.After(ti)
}
func (t *Table) GetSavedFilename(rt ReportType, ti time.Time) string {
	var last, n int
	var buf []byte
	for {
		l := strings.IndexRune(t.savefile[last:], '{')
		if l == -1 {
			break
		}
		l += last + 1
		r := strings.IndexRune(t.savefile[l:], '}')
		if r == -1 {
			break
		}
		r += l
		if buf == nil {
			buf = make([]byte, len(t.savefile))
		}
		if r - l == 0 {
			continue
		}
		n += copy(buf[n:], t.savefile[last:l-1])
		switch t.savefile[l:r] {
		case "type":
			n += copy(buf[n:], rt.String())
		default:
			n += copy(buf[n:], ti.Format(t.savefile[l:r]))
		}
		last = r+1
	}
	if buf != nil {
		n += copy(buf[n:], t.savefile[last:])
		return string(buf[:n])
	}
	return t.savefile
}
