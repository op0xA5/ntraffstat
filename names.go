package main

import(
	"sync"
	"io"
)

type Name struct {
	p *string
}
type NamePool struct {
	m    map[string]Name
	lock sync.RWMutex
}
func NewNamePool(_cap int) *NamePool {
	return &NamePool{
		m: make(map[string]Name, _cap),
	}
}
func (np *NamePool) Get(v string) Name {
	if v == "" {
		return EmptyStringName
	}

	np.lock.RLock()
	name, _ := np.m[v]
	np.lock.RUnlock()
	return name
}
func (np *NamePool) Put(v string) Name {
	if v == "" {
		return EmptyStringName
	}
	np.lock.RLock()
	name, ok := np.m[v]
	if !ok {
		np.lock.RUnlock()
		np.lock.Lock()
		name = Name{&v}
		np.m[v] = name
		np.lock.Unlock()
	} else {
		np.lock.RUnlock()
	}
	return name
}
func (np *NamePool) Len() int {
	np.lock.RLock()
	n := len(np.m)
	np.lock.RUnlock()
	return n
}
func (np *NamePool) Empty() {
	np.lock.Lock()
	count := len(np.m)
	np.m = make(map[string]Name, count)
	np.lock.Unlock()
}
func (n Name) IsNil() bool {
	return n.p == nil
}
func (n Name) String() string {
	return *n.p
}
var emptyString = "";
var EmptyStringName = Name{ &emptyString }

type NameList []Name
func (nl NameList) WriteJson(w io.Writer) error {
	var err error	
	if _ , err = w.Write([]byte{ '[' }); err != nil {
		return err
	}
	for i, item := range nl {
		if i > 0 {
			if _, err = w.Write(comma); err != nil {
				return err
			}
		}
		if err = encodeString(w, item.String()); err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{ ']' })
	return err
}
