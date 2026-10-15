package connection

import (
	"sync"
	"time"

	"relay/sdk"
)

type Connection struct {
	ID          uint64
	RemoteAddr  string
	Protocol    int
	State       sdk.State
	Player      sdk.Player
	HasPlayer   bool
	Backend     string
	ConnectedAt uint64
}

type Manager struct {
	mu     sync.RWMutex
	byID   map[uint64]Connection
	nextID uint64
}

func NewManager() *Manager {
	return &Manager{byID: make(map[uint64]Connection)}
}

var DefaultManager = NewManager()

func (m *Manager) Add(remoteAddr string) Connection {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	c := Connection{
		ID:          m.nextID,
		RemoteAddr:  remoteAddr,
		State:       sdk.StateHandshake,
		ConnectedAt: uint64(time.Now().UnixMilli()),
	}
	m.byID[c.ID] = c
	return c
}

func (m *Manager) Get(id uint64) (Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	c, ok := m.byID[id]
	return c, ok
}

func (m *Manager) SetProtocol(id uint64, protocol int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.byID[id]
	if !ok {
		return false
	}

	c.Protocol = protocol
	m.byID[id] = c
	return true
}

func (m *Manager) SetState(id uint64, s sdk.State) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.byID[id]
	if !ok {
		return false
	}

	c.State = s
	m.byID[id] = c
	return true
}

func (m *Manager) SetPlayer(id uint64, p sdk.Player) bool {
	if err := p.Validate(); err != nil {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.byID[id]
	if !ok {
		return false
	}

	c.Player = p
	c.HasPlayer = true
	m.byID[id] = c
	return true
}

func (m *Manager) SetBackend(id uint64, backend string) bool {
	if backend == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.byID[id]
	if !ok {
		return false
	}

	c.Backend = backend
	m.byID[id] = c
	return true
}

func (m *Manager) Remove(id uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.byID[id]; !ok {
		return false
	}

	delete(m.byID, id)
	return true
}

func (m *Manager) List() []Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Connection, 0, len(m.byID))
	for _, c := range m.byID {
		out = append(out, c)
	}
	return out
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.byID)
}

func (m *Manager) CountInState(s sdk.State) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, c := range m.byID {
		if c.State == s {
			n++
		}
	}
	return n
}
