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
	"strings"
	"encoding/json"
	"bytes"
)

const ConfigFilename string = "config.json"

var pidFile string = os.Args[0] + ".pid"
type ConfigType struct {
	HttpListen          string

	FifoFile            string
	RecreateIfNotFifo   bool
//	CmdAfterFifoCreated     string

	FileRoot            string

	RecordFullUrl       bool
	RecordRefer         bool
}
var Config = ConfigType{
	HttpListen:    ":8091",

	FifoFile:            "/tmp/nginx_traffic.log",
	RecreateIfNotFifo:   false,
//	CmdAfterFifoCreated: "",

	RecordFullUrl: true,
	RecordRefer:   true,
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime)

	err := LoadConfig()
	if err != nil {
		log.Printf("failed load config: %v\n", err)
		return
	}

	initWorker()

	Config.FileRoot = strings.TrimSuffix(Config.FileRoot, "/")

	if Config.FifoFile != "" {
		fifo, err := OpenFifo(Config.FifoFile, Config.RecreateIfNotFifo)
		if err != nil {
			log.Printf("error open fifo: %v\n", err)
			return
		}

		loadWorkerStaging()

		go workLoop()
		go readLoop(fifo)
	} else {
		log.Println("fifo file not set, start as log reader mode")
	}

	ln, err := net.Listen("tcp", Config.HttpListen)
	if err != nil {
		log.Printf("failed listen tcp %s, http server offline\n", Config.HttpListen)
	} else {
		log.Printf("start http server at %s\n", Config.HttpListen)
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

func LoadConfig() error {
	f, err := os.OpenFile(ConfigFilename, os.O_RDWR, 0660)
	if os.IsNotExist(err) {
		f, err = os.Create(ConfigFilename)
		if err != nil {
			return err
		}
		goto write
	}
	if err != nil {
		return err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&Config)
	if err != nil {
		return err
	}

write:
	b, err := json.Marshal(&Config)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "\t")
	if err != nil {
		return err
	}
	if _, err = f.Seek(0, 0); err != nil {
		return err
	}
	if err = f.Truncate(0); err != nil {
		return err
	}
	_, err = out.WriteTo(f)
	return err
}
