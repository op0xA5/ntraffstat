package main

import(
	"os"
	"path"
	"encoding/json"
	"time"
	"strconv"
	"errors"
	"log"
)

const STAGING_DIR = "staging"

func makeStaging(t *Table, prefix string) {
	os.Mkdir(STAGING_DIR, 0750)

	if err := stagingReport(t, REPORT_IP, path.Join(STAGING_DIR, prefix+"ip")); err != nil {
		log.Printf("error staging %sip: %v\n", prefix, err)
	}	
	if err := stagingReport(t, REPORT_HOST, path.Join(STAGING_DIR, prefix+"host")); err != nil {
		log.Printf("error staging %shost: %v\n", prefix, err)
	}	
	if err := stagingReport(t, REPORT_PATH, path.Join(STAGING_DIR, prefix+"path")); err != nil {
		log.Printf("error staging %spath: %v\n", prefix, err)
	}	
	if err := stagingReport(t, REPORT_URL, path.Join(STAGING_DIR, prefix+"url")); err != nil {
		log.Printf("error staging %surl: %v\n", prefix, err)
	}	
	if err := stagingReport(t, REPORT_FILE, path.Join(STAGING_DIR, prefix+"file")); err != nil {
		log.Printf("error staging %sfile: %v\n", prefix, err)
	}
}

func stagingReport(t *Table, rt ReportType, file string) error {
	r := t.GetCurrentSnapshot(rt)
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.WriteJson(f)
}

func loadStaging(t *Table, prefix string, np_ip, np_url, np_file *NamePool) {
	var rs *ReportSnapshot
	var err error
	rs, err = loadSnapshot(path.Join(STAGING_DIR, prefix+"ip"), np_ip)
	if rs != nil {
		err = tableFillSnapshot(t, REPORT_IP, rs)
	}
	if err != nil {
		log.Printf("error load staging %sip: %v\n", prefix, err)
	}
	rs, err = loadSnapshot(path.Join(STAGING_DIR, prefix+"host"), np_url)
	if rs != nil {
		err = tableFillSnapshot(t, REPORT_HOST, rs)
	}
	if err != nil {
		log.Printf("error load staging %shost: %v\n", prefix, err)
	}
	rs, err = loadSnapshot(path.Join(STAGING_DIR, prefix+"path"), np_url)
	if rs != nil {
		err = tableFillSnapshot(t, REPORT_PATH, rs)
	}
	if err != nil {
		log.Printf("error load staging %spath: %v\n", prefix, err)
	}
	rs, err = loadSnapshot(path.Join(STAGING_DIR, prefix+"url"), np_url)
	if rs != nil {
		err = tableFillSnapshot(t, REPORT_URL, rs)
	}
	if err != nil {
		log.Printf("error load staging %surl: %v\n", prefix, err)
	}
	rs, err = loadSnapshot(path.Join(STAGING_DIR, prefix+"file"), np_file)
	if rs != nil {
		err = tableFillSnapshot(t, REPORT_FILE, rs)
	}
	if err != nil {
		log.Printf("error load staging %sfile: %v\n", prefix, err)
	}
}

var ErrTimespanNotAcceptable = errors.New("timespan not acceptable")
func tableFillSnapshot(t *Table, rt ReportType, rs *ReportSnapshot) error {
	if rs.end.Before(rs.begin) ||
		rs.begin.Before(t.last) ||
		t.next.Before(rs.end) {
		return ErrTimespanNotAcceptable
	}
	m := NewReport(rt, 0)
	m.loadSnapshotData(rs)
	switch rt {
	case REPORT_IP:
		t.ip = m
	case REPORT_HOST:
		t.host = m
	case REPORT_PATH:
		t.path = m
	case REPORT_URL:
		t.url = m
	case REPORT_FILE:
		t.file = m
	default:
		panic("unexpected report type")
	}
	if t.begin.After(rs.begin) {
		t.begin = rs.begin
	}
	return nil
}

const(
	FileStructReqBody     = 0
	FileStructReqDreqBody = 1
)
func loadSnapshot(filename string, np *NamePool) (*ReportSnapshot, error) {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}
	defer f.Close()

	snap := ReportSnapshot{
		filename: filename,
	}

	d := json.NewDecoder(f)
	d.UseNumber()

	if err := jsonTestDelim(d, '{'); err != nil {
		return nil, err
	}

	var str string
	var length int
	var fieldStruct int
	for d.More() {
		if str, err = jsonGetString(d); err != nil {
			return nil, err
		}
		switch str {
		case "begin":
			if str, err = jsonGetString(d); err != nil {
				return nil, err
			}
			snap.begin, err = time.Parse(time.RFC3339, str)
			if err != nil {
				return nil, err
			}
		case "end":
			if str, err = jsonGetString(d); err != nil {
				return nil, err
			}
			snap.end, err = time.Parse(time.RFC3339, str)
			if err != nil {
				return nil, err
			}
		case "length":
			if str, err = jsonGetNumber(d); err != nil {
				return nil, err
			}
			length, err = strconv.Atoi(str)
			if err != nil {
				return nil, err
			}
		case "fields":
			r := new(json.RawMessage)
			if err = d.Decode(r); err != nil {
				return nil, err
			}
			if string(*r) == `["req","body"]` {
				fieldStruct = FileStructReqBody		
			} else if string(*r) == `["req","dreq","body"]` {
				fieldStruct = FileStructReqDreqBody
			} else {
				return nil, ErrBadFieldsSet
			}
		case "names":
			if err = jsonTestDelim(d, '['); err != nil {
				return nil, err
			}
			if length == 0 && d.More() {
				return nil, ErrLengthNotMatch
			}
			if snap.items == nil {
				snap.items = make([]nameDataPair, length)
			}
			var n int
			for d.More() {
				if n >= len(snap.items) {
					return nil, ErrLengthNotMatch
				}

				if str, err = jsonGetString(d); err != nil {
					return nil, err
				}
				snap.items[n].name = np.Put(str)
				n++
			}
			if err = jsonTestDelim(d, ']'); err != nil {
				return nil, err
			}

		case "data":
			if err = jsonTestDelim(d, '['); err != nil {
				return nil, err
			}
			if length == 0 && d.More() {
				return nil, ErrLengthNotMatch
			}
			if snap.items == nil {
				snap.items = make([]nameDataPair, length)
			}

			var n int
			var rd ReportData
			if fieldStruct == FileStructReqBody {
				for d.More() {
					if n >= len(snap.items) {
						return nil, ErrLengthNotMatch
					}
					
					if str, err = jsonGetNumber(d); err != nil {
						return nil, err
					}
					if rd.Req, err = strconv.ParseUint(str, 10, 64); err != nil {
						return nil, err
					}
					if str, err = jsonGetNumber(d); err != nil {
						return nil, err
					}
					if rd.Body, err = strconv.ParseUint(str, 10, 64); err != nil {
						return nil, err
					}
					rd.DataReq = rd.Req
					snap.items[n].data = rd
					n++
				}
			} else if fieldStruct == FileStructReqDreqBody {
				for d.More() {
					if n >= len(snap.items) {
						return nil, ErrLengthNotMatch
					}

					if str, err = jsonGetNumber(d); err != nil {
						return nil, err
					}
					if rd.Req, err = strconv.ParseUint(str, 10, 64); err != nil {
						return nil, err
					}
					if str, err = jsonGetNumber(d); err != nil {
						return nil, err
					}
					if rd.DataReq, err = strconv.ParseUint(str, 10, 64); err != nil {
						return nil, err
					}
					if str, err = jsonGetNumber(d); err != nil {
						return nil, err
					}
					if rd.Body, err = strconv.ParseUint(str, 10, 64); err != nil {
						return nil, err
					}
					snap.items[n].data = rd
					n++
				}
			} else {
				panic("bad field struct")
			}

			if err = jsonTestDelim(d, ']'); err != nil {
				return nil, err
			}
		default:
			t, err := d.Token()
			if err != nil {
				return nil, ErrBadFormat
			}
			if _, isDelim := t.(json.Delim); isDelim {
				return nil, ErrBadFormat
			}
		}
	}
	if err = os.Remove(filename); err != nil {
		log.Printf("error delete staging file: %s %v\n", filename, err)
	}
	return &snap, nil
}
