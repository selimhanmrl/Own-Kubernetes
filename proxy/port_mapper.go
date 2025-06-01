package proxy

import (
	"math/rand"
	"sync"
)

type PortMapper struct {
	usedPorts map[int]bool
	mutex     sync.Mutex
}

func NewPortMapper() *PortMapper {
	return &PortMapper{
		usedPorts: make(map[int]bool),
	}
}

func (pm *PortMapper) AssignPort() int {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for {
		port := 30000 + rand.Intn(2768)
		if !pm.usedPorts[port] {
			pm.usedPorts[port] = true
			return port
		}
	}
}
