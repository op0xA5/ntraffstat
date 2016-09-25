package main

import (
	"time"
	"io"
	"strings"
	"strconv"
	"errors"
	"os"
	"compress/gzip"
	"encoding/json"
	"fmt"
)

type Trend struct {
	items     []ReportData

	update   time.Duration
	split    Truncater
	next     time.Time
	begin    time.Time
	end      time.Time

	current  int

	savefile string
	w        *DumpWorker
}
func NewTrend(now time.Time, update time.Duration, split Truncater) *Trend {
	t := &Trend{
		update:   update,
		split:    split,
	}
	t.init(now)
	return t
}
func (t *Trend) init(now time.Time) {
	t.next = DurationTruncater(t.update).Truncate(now).Add(t.update)
	var ds time.Duration
	t.begin = t.split.Truncate(now)
	ds      = t.split.Duration(now)
	t.end   = t.begin.Add(ds)

	count := int(ds / t.update)
	if ds % t.update > 0 {
		count++
	}
	t.items   = make([]ReportData, count)
	t.current = int(now.Sub(t.begin) / t.update)
}
func (t *Trend) SetSavefile(filename string, w *DumpWorker) {
	t.savefile = filename
	t.w = w
}
func (t *Trend) Add(time time.Time, data ReportData) {
	if !time.Before(t.end) {
		t.Flush()
		t.init(time)
	}
	for !time.Before(t.next) {
		t.Flush()
	}

	t.items[t.current].Req  += data.Req
	t.items[t.current].Body += data.Body
}
func (t *Trend) Flush() {
	t.next = t.next.Add(t.update)
	t.current++

	if t.w != nil {
		t.w.Add(&TrendSnapshot{
			filename: t.GetSavedFilename(t.begin),
			begin: t.begin, end: t.end,
			items: t.items,
		})
	}
}
var ErrBadFormat        = errors.New("bad format")
var ErrTimespanNotMatch = errors.New("timespan not match")
var ErrBadFieldsSet     = errors.New("bad fields set")
var ErrLengthNotMatch   = errors.New("length not match")
func (t *Trend) Load(filename string, useGzip bool) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	r := io.Reader(f)
	if useGzip {
		r, err = gzip.NewReader(f)
		if err != nil {
			return err
		}
	}
	d := json.NewDecoder(r)
	d.UseNumber()

	if err := jsonTestDelim(d, '{'); err != nil {
		return err
	}
	var begin, end time.Time
	var length int
	var str string
	for d.More() {
		if str, err = jsonGetString(d); err != nil {
			return err
		}
		switch str {
		case "begin":
			if str, err = jsonGetString(d); err != nil {
				return err
			}
			begin, err = time.Parse(time.RFC3339, str)
			if err != nil {
				return err
			}
			if begin != t.begin {
				return ErrTimespanNotMatch
			}
		case "end":
			if str, err = jsonGetString(d); err != nil {
				return err
			}
			end, err = time.Parse(time.RFC3339, str)
			if err != nil {
				return err
			}
			if end != t.end {
				return ErrTimespanNotMatch
			}
		case "length":
			if str, err = jsonGetNumber(d); err != nil {
				return err
			}
			length, err = strconv.Atoi(str)
			if err != nil {
				return err
			}
			if length != len(t.items) {
				return ErrLengthNotMatch
			}
		case "fields":
			r := new(json.RawMessage)
			if err = d.Decode(r); err != nil {
				return err
			}
			if string(*r) != `["req","body"]` {
				return ErrBadFieldsSet
			}
		case "data":
			if err = jsonTestDelim(d, '['); err != nil {
				return err
			}
			if length != len(t.items) {
				return ErrLengthNotMatch
			}
			var n int
			for d.More() {
				if n >= len(t.items) {
					return ErrLengthNotMatch
				}

				var rd ReportData
				if str, err = jsonGetNumber(d); err != nil {
					return err
				}
				if rd.Req, err = strconv.ParseUint(str, 10, 64); err != nil {
					return err
				}
				if str, err = jsonGetNumber(d); err != nil {
					return err
				}
				if rd.Body, err = strconv.ParseUint(str, 10, 64); err != nil {
					return err
				}
				t.items[n] = rd
				n++
			}
			if err = jsonTestDelim(d, ']'); err != nil {
				return err
			}
		default:
			t, err := d.Token()
			if err != nil {
				return ErrBadFormat
			}
			if _, isDelim := t.(json.Delim); isDelim {
				return ErrBadFormat
			}
		}
	}
	return nil
}

func (t *Trend) GetCurrentFilename() string {
	return t.GetSavedFilename(t.begin)
}
func (t *Trend) GetSavedFilename(ti time.Time) string {
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
		n += copy(buf[n:], ti.Format(t.savefile[l:r]))
		last = r+1
	}
	if buf != nil {
		n += copy(buf[n:], t.savefile[last:])
		return string(buf[:n])
	}
	return t.savefile
}

type TrendSnapshot struct {
	filename      string
	begin, end    time.Time
	items         []ReportData
}
func (ts *TrendSnapshot) Filename() string {
	return ts.filename
}
func (ts *TrendSnapshot) WriteJson(w io.Writer) error {
	var err error
	if _, err = fmt.Fprintf(w,
		`{"begin":"%s","end":"%s","duration":%g,"length":%d,"fields":["req","body"],"data":[`,
		ts.begin.Format(time.RFC3339), ts.end.Format(time.RFC3339),
		ts.end.Sub(ts.begin).Seconds(), len(ts.items)); err != nil {
		return err
	}
	var buffer [24]byte
	for i, item := range ts.items {
		if i > 0 {
			w.Write(comma)
		}
		w.Write(strconv.AppendUint(buffer[:0], item.Req, 10))
		w.Write(comma)
		w.Write(strconv.AppendUint(buffer[:0], item.Body, 10))	
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{ ']', '}' })
	return err
}

func jsonTestDelim(d *json.Decoder, want json.Delim) error {
	t, err := d.Token()
	if err != nil {
		if err == io.EOF {
			return ErrBadFormat
		}
		return err
	}
	if t != want {
		return ErrBadFormat
	}
	return nil
}
func jsonGetNumber(d *json.Decoder) (string, error) {
	t, err := d.Token()
	if err != nil {
		if err == io.EOF {
			return "", ErrBadFormat
		}
		return "", err
	}
	if str, ok := t.(json.Number); ok {
		return string(str), nil
	}
	return "", ErrBadFormat
}
func jsonGetString(d *json.Decoder) (string, error) {
	t, err := d.Token()
	if err != nil {
		if err == io.EOF {
			return "", ErrBadFormat
		}
		return "", err
	}
	if str, ok := t.(string); ok {
		return str, nil
	}
	return "", ErrBadFormat
}
