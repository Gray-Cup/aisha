package orchestrator

import (
	"fmt"
	"net"
	"sync"
)

type portAllocator struct {
	mu    sync.Mutex
	inUse map[int]bool
}

func newPortAllocator(reserved []int) *portAllocator {
	pa := &portAllocator{inUse: make(map[int]bool)}
	for _, p := range reserved {
		pa.inUse[p] = true
	}
	return pa
}

func (pa *portAllocator) Allocate() (int, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	for p := 4000; p <= 4999; p++ {
		if !pa.inUse[p] && isPortFree(p) {
			pa.inUse[p] = true
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free ports in range 4000–4999")
}

func (pa *portAllocator) Reserve(port int) {
	pa.mu.Lock()
	pa.inUse[port] = true
	pa.mu.Unlock()
}

func (pa *portAllocator) Free(port int) {
	pa.mu.Lock()
	delete(pa.inUse, port)
	pa.mu.Unlock()
}

func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
