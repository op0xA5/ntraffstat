package main

import (
	"os"
	"io"
	"log"
	"compress/gzip"
	"runtime"
	"path"
	"time"
)

type DumpTask interface {
	Filename() string
	WriteJson(w io.Writer) error
}
type DumpWorker struct {
	base      string
	mode      os.FileMode
	GzipLevel int

	logger    *log.Logger
	enableLog bool

	task      chan<- DumpTask
}
func NewDumpWorker(base string, mode os.FileMode) *DumpWorker {
	rc := make(chan DumpTask, 512)
	rdw := &DumpWorker{
		base: base,
		mode: mode,
		task: rc,
	}
	go rdw.loop(rc)
	return rdw
}
func (rdw *DumpWorker) SetLogger(logger *log.Logger) {
	rdw.enableLog = true
	rdw.logger    = logger
}
func (rdw *DumpWorker) loop(rc <-chan DumpTask) {
	runtime.LockOSThread()
	for {
		select {
		case r, more := <- rc:
			if !more {
				return
			}

			rdw.processTask(r)
		}
	}
}
func (rdw *DumpWorker) processTask(rdt DumpTask) error {
	var err error
	filename := rdt.Filename()

	if rdw.enableLog {
		defer func() {
			if err == nil {
				if rdw.logger != nil {
					rdw.logger.Printf("file saved: %s\n", filename)
				} else {
					log.Printf("file saved: %s\n", filename)
				}
			} else {
				if rdw.logger != nil {
					rdw.logger.Printf("failed save file: %s  %v\n", filename, err)
				} else {
					log.Printf("failed save file: %s  %v\n", filename, err)
				}
			}
		}()
	}

	file := rdw.FullPath(filename)
	dir := path.Dir(file)
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, rdw.mode | 0111)
		if err != nil {
			return err
		}
	}
	var f *os.File
	f, err = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, rdw.mode)
	if err != nil {
		return err
	}
	defer f.Close()
	var w io.Writer = f
	if rdw.GzipLevel > 0 {
		var gz *gzip.Writer
		gz, err = gzip.NewWriterLevel(f, rdw.GzipLevel)
		if err != nil {
			return err
		}
		defer gz.Close()
		w = gz
	}
	err = rdt.WriteJson(w)
	return err
}
func (rdw *DumpWorker) Add(rdt DumpTask) {
	rdw.task <- rdt
}
func (rdw *DumpWorker) Exit() {
	close(rdw.task)
	for len(rdw.task) != 0 {
		time.Sleep(1 * time.Millisecond)
	}
}
func (rdw *DumpWorker) FullPath(file string) string {
	return path.Join(rdw.base, file)
}