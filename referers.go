package main

import (
	"sync"
)

const MAX_REFERER_RECORDS = 16

type RefererTable struct {
	m    map[Name][]Name
	lock sync.RWMutex
}
func NewRefererTable(_cap int) *RefererTable {
	return &RefererTable{
		m: make(map[Name][]Name, _cap),
	}
}
func (rt *RefererTable) Add(name, to Name) {
	if to.IsNil() {
		return
	}

	rt.lock.RLock()
	tos, ok := rt.m[name]
	if !ok {
		rt.lock.RUnlock()
		rt.lock.Lock()
		tos = make([]Name, MAX_REFERER_RECORDS)
		tos[0] = to
		rt.m[name] = tos
		rt.lock.Unlock()
	} else if tos[0] != to {
		rt.lock.RUnlock()
		rt.lock.Lock()
		var i int
		for i = 1; i < MAX_REFERER_RECORDS-1; i++ {
			if tos[i] == to || tos[i].IsNil() {
				break
			}
		}
		copy(tos[1:], tos[:i])
		tos[0] = to
		rt.lock.Unlock()
	} else {
		rt.lock.RUnlock()
	}
}
func (rt *RefererTable) Find(name Name) NameList {
	var out NameList
	rt.lock.RLock()
	tos, ok := rt.m[name]
	if ok {
		out = make(NameList, len(tos))
		copy(out, tos)
	}	
	rt.lock.RUnlock()
	for i := range out {
		if out[i].IsNil() {
			out = out[:i]
			break
		}
	}
	return out
}
func (rt *RefererTable) Empty() {
	rt.lock.Lock()
	count := len(rt.m)
	rt.m = make(map[Name][]Name, count)
	rt.lock.Unlock()
}