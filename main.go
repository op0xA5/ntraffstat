package main

import (
	"log"
	"os"
	"sync"
	"io/ioutil"
	"strconv"
	"os/signal"
	"syscall"
	"net"
)
var httpAddr string = ":8091"
var pidFile string = os.Args[0] + ".pid"

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime)

	initWorker()

	loadWorkerStaging()

	startWorker()

	ln, err := net.Listen("tcp", httpAddr)
	if err != nil {
		log.Printf("failed listen tcp %s, http server offline\n", httpAddr)
	} else {
		log.Printf("start http server at %s\n", httpAddr)
		go startHttpServer(ln)
	}

	ioutil.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0660)
	defer os.Remove(pidFile)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGQUIT)
	for sig := range sigChan {
		if s, ok := sig.(syscall.Signal); ok {
			switch s {
			case syscall.SIGQUIT:
				ln.Close()
				log.Printf("http server offline\n")
				exitWorker()

				wg := new(sync.WaitGroup)
				wg.Add(3)
				go func() {
					defer func() {
						wg.Done()
					}()
					makeStaging(tbl_m5, "m5_")
				}()
				go func() {
					defer func() {
						wg.Done()
					}()
					makeStaging(tbl_h, "h_")
				}()
				go func() {
					defer func() {
						wg.Done()
					}()
					makeStaging(tbl_d, "d_")
				}()
				wg.Wait()
				log.Printf("finish staging, good bye\n")

				os.Exit(0)
			}
		}
	}
}
