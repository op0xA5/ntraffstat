package main

import (
	"io"
	"os"
	"syscall"
	"log"
)

func OpenFifo(file string) (io.ReadCloser, error) {
	r, isFifo, err := _openFifo(file)
	if err != nil {
		return nil, err
	}
	if !isFifo {
		log.Printf("warn %s is not fifo\n", file)
	}
	if isFifo {
		return &reopenReader{ file, r }, nil
	}
	return r, nil
}
func _openFifo(file string) (r io.ReadCloser, isFifo bool, err error) {
	var fi os.FileInfo
	fi, err = os.Stat(file)
	if os.IsNotExist(err) {
		err = syscall.Mkfifo(file, 0660)
		isFifo = true
	}
	if err != nil {
		return
	}
	if fi != nil {
		isFifo = fi.Mode() & (os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice) != 0
	}
	if isFifo {
		r, err = os.OpenFile(file, os.O_RDONLY, os.ModeNamedPipe)
	} else {
		r, err = os.Open(file)
	}
	return
}
type reopenReader struct {
	file string
	r    io.ReadCloser
}
func (rr *reopenReader) Read(p []byte) (n int, err error) {
	if rr.r == nil {
		return 0, io.ErrClosedPipe
	}
	n, err = rr.r.Read(p)
	if err == io.EOF {
		log.Printf("fifo eof occurred, try reopen\n")
		err = rr.r.Close()
		if err != nil {
			log.Printf("fifo error close file: %v\n", err)
		}
		rr.r, _, err = _openFifo(rr.file)
		if err != nil {
			log.Printf("fifo error reopen: %v\n", err)
			return n, err
		}
	}
	return
}
func (rr *reopenReader) Close() error {
	var err error
	if rr.r != nil {
		err = rr.r.Close()
		rr.r = nil
	}
	return err
}
