package main

import(
	"time"
	"runtime"
	"io"
	"path"
	"log"
	"strings"
)

var tbl_m5  *Table
var tbl_h   *Table
var tbl_d   *Table

var np_ip   *NamePool
var np_url  *NamePool
var np_file *NamePool

var ref_url_refer  *RefererTable
var ref_path_refer *RefererTable
var ref_file_path  *RefererTable

var tr_m *Trend
var tr_h *Trend

var dump_worker *DumpWorker

func initWorker() {
	now := time.Now()

	np_ip   = NewNamePool(4096)
	np_url  = NewNamePool(128 * 1024)
	np_file = NewNamePool(128 * 1024)

	dump_worker = NewDumpWorker("logs", 0666)
	dump_worker.SetLogger(nil)
	dump_worker.GzipLevel = 9


	tbl_m5  = NewTable(now, DurationTruncater(5 * time.Minute))
	tbl_m5.SetSavefile("{2006/0102}/m5-{1504}-{type}.json.gz", dump_worker)
	tbl_h = NewTable(now, DurationTruncater(1 * time.Hour))
	tbl_h.SetSavefile("{2006/0102}/h-{15}-{type}.json.gz", dump_worker)
	tbl_d = NewTable(now, DurationTruncater(24 * time.Hour))
	tbl_d.SetSavefile("{2006/0102}/d-{type}.json.gz", dump_worker) 

	ref_url_refer  = NewRefererTable(128 * 1024)
	ref_path_refer = NewRefererTable(128 * 1024)
	ref_file_path   = NewRefererTable(128 * 1024)

	tr_m = NewTrend(now, 1 * time.Minute, DurationTruncater(24 * time.Hour))
	tr_m.SetSavefile("{2006/0102}/tr_m.json.gz", dump_worker)
	if err := tr_m.Load(path.Join("logs", tr_m.GetCurrentFilename()), true); err != nil {
		log.Printf("failed load trend_m: %s %v\n", tr_m.GetCurrentFilename(), err)
	} else {
		log.Printf("trend_m loaded\n")
	}
	tr_h = NewTrend(now, 1 * time.Hour, MonthTruncater{})
	tr_h.SetSavefile("{2006/01}01/tr_h.json.gz", dump_worker)
	if err := tr_h.Load(path.Join("logs", tr_h.GetCurrentFilename()), true); err != nil {
		log.Printf("failed load trend_h: %s %v\n", tr_h.GetCurrentFilename(), err)
	} else {
		log.Printf("trend_h loaded\n")
	}
}

func loadWorkerStaging() {
	loadStaging(tbl_m5, "m5_", np_ip, np_url, np_file)
	loadStaging(tbl_h, "h_", np_ip, np_url, np_file)
	loadStaging(tbl_d, "d_", np_ip, np_url, np_file)
	log.Println("staging loaded")
}

var workRC   = make(chan RawRecord, 512)
var workExit bool
var workFifo io.ReadCloser
func exitWorker() {
	workExit = true
	close(workRC)
	for len(workRC) != 0 {
		runtime.Gosched()
	}
	tr_m.Flush()
	tr_h.Flush()
	dump_worker.Exit()
}

func workLoop() {
	runtime.LockOSThread()

	ti := time.NewTicker(50 * time.Millisecond)
	defer ti.Stop()

	update  := DurationTruncater(24 * time.Hour)
	next, _ := TruncateNext(update, time.Now())

	for {
		select {
		case rr, more := <- workRC:
			if !more {
				return
			}

			if !rr.Time.Before(next) {
				ref_url_refer.Empty()
				ref_path_refer.Empty()
				ref_file_path.Empty()
				np_ip.Empty()
				np_url.Empty()
				np_file.Empty()

				next = next.Add(update.Duration(next))
			}

			if !(rr.File == "" || rr.File == "-" ||
				strings.HasPrefix(rr.File, Config.FileRoot)) {
				continue
			}
			rr.Refer = strings.TrimPrefix(rr.Refer, "http://")

			rr.File = rr.File[len(Config.FileRoot):]
			rec := Record{
				Time:  rr.Time,
				Ip:    np_ip.Put(rr.Ip),
				Host:  np_url.Put(rr.Host),
				Path:  np_url.Put(rr.Path),
				Url:   np_url.Put(rr.Url),
				File:  np_file.Put(rr.File),
				Body:  rr.Body,
				Refer: np_url.Put(rr.Refer),
			}

			tbl_m5.Add(rec)
			tbl_h.Add(rec)
			tbl_d.Add(rec)
			if Config.RecordRefer {
				ref_url_refer.Add(rec.Url, rec.Refer)
				ref_path_refer.Add(rec.Path, rec.Refer)
				ref_file_path.Add(rec.File, rec.Path)
			}
			tr_m.Add(rec.Time, ReportData{ Req: 1, Body: rec.Body })
			tr_h.Add(rec.Time, ReportData{ Req: 1, Body: rec.Body })
			
		case now := <- ti.C:
			tbl_m5.Flush(now)
			tbl_h.Flush(now)
			tbl_d.Flush(now)
			tr_m.Add(now, ReportData{})
			tr_h.Add(now, ReportData{})
		}
	}
}

func readLoop(fifo io.ReadCloser) {
	defer fifo.Close()

	rr := NewRecordReader(fifo)
	if !Config.RecordFullUrl {
		rr.AddFlag(StripFullUrl)
	}
	if !Config.RecordRefer {
		rr.AddFlag(NoRefer)
	}
	for rr.Scan() {
		if workExit {
			return
		}
		if rr.Valid() {
			workRC <- rr.RawRecord
		} else if len(rr.Bytes()) != 0 {
			log.Printf("error log format '%s'\n", rr.Bytes())
		}
	}
	if rr.Err() != nil {
		log.Printf("error read fifo: %v\n", rr.Err())
	}
}
