package main

import (
	"time"
)

type Truncater interface {
	Truncate(t time.Time) time.Time
	Duration(t time.Time) time.Duration
}

type DurationTruncater time.Duration
func (dt DurationTruncater) Truncate(t time.Time) time.Time {
	_, offset := t.Zone()
	fix := time.Duration(offset) * time.Second
	return t.Add(+fix).Truncate(time.Duration(dt)).Add(-fix)
}
func (dt DurationTruncater) Duration(t time.Time) time.Duration {
	return time.Duration(dt)
}

type YearTruncater struct{}
func (yt YearTruncater) Truncate(t time.Time) time.Time {
	year, _, _ := t.Date()
	return time.Date(year, time.January, 1, 0, 0, 0, 0, t.Location())
}
func (yt YearTruncater) Duration(t time.Time) time.Duration {
	year, _, _ := t.Date()
	return time.Date(year+1, time.January, 1, 0, 0, 0, 0, t.Location()).Sub(
		time.Date(year, time.January, 1, 0, 0, 0, 0, t.Location()))
}

type MonthTruncater struct{}
func (yt MonthTruncater) Truncate(t time.Time) time.Time {
	year, month, _ := t.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, t.Location())
}
func (yt MonthTruncater) Duration(t time.Time) time.Duration {
	year, month, _ := t.Date()
	return time.Date(year, month+1, 1, 0, 0, 0, 0, t.Location()).Sub(
		time.Date(year, month, 1, 0, 0, 0, 0, t.Location()))
}

func TruncateNext(tr Truncater, t time.Time) (time.Time, time.Duration) {
	d := tr.Duration(t)
	return tr.Truncate(t).Add(d), d
}
