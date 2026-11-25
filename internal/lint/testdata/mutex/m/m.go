package m

import "sync"

type Bad struct {
	Mu    sync.Mutex
	Ready chan bool
	WG    *sync.WaitGroup
	sync.RWMutex
}

type Good struct {
	mu   sync.Mutex
	done chan struct{}
	Name string
}

type hidden struct {
	Mu sync.Mutex
}

func (g *Good) Wait() { g.mu.Lock(); g.mu.Unlock(); <-g.done }
