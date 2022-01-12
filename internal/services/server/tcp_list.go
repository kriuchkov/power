package server

import (
	"net"
	"strings"
	"sync"
)

// powTCPList is a structure with list tcp addresses
type powTCPList struct {
	mx   sync.Mutex
	data []*net.TCPAddr
	used map[*net.TCPAddr]struct{}
}

func (l *powTCPList) String() string {
	out := []string{}
	for i := range l.data {
		out = append(out, l.data[i].String())
	}
	return strings.Join(out, ", ")
}

func (list *powTCPList) Get() (*net.TCPAddr, error) {
	for i := range list.data {
		if _, ok := list.used[list.data[i]]; !ok {
			list.mx.Lock()
			defer list.mx.Unlock()
			list.used[list.data[i]] = struct{}{}
			return list.data[i], nil
		}
	}
	return nil, ErrTCPListEmpty
}

func (l *powTCPList) Free(key *net.TCPAddr) {
	l.mx.Lock()
	defer l.mx.Unlock()
	delete(l.used, key)
}

func newPowTCPList(d []*net.TCPAddr) *powTCPList {
	return &powTCPList{
		data: d,
		used: make(map[*net.TCPAddr]struct{}),
		mx:   sync.Mutex{},
	}
}
