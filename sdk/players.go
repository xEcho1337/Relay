package sdk

import (
	"errors"
	"sync"
)

// Bounds shared with the WASM ABI.
const (
	MaxUsernameLen = 16
	MaxHostLen     = 253
	MaxReasonLen   = 256
)

type Player struct {
	Username    string
	UUID        string
	Backend     string
	ClientIP    string
	ConnectedAt uint64
}

func (p Player) Validate() error {
	if len(p.Username) == 0 || len(p.Username) > MaxUsernameLen {
		return errors.New("sdk: invalid username length")
	}
	if p.UUID == "" {
		return errors.New("sdk: empty UUID")
	}
	return nil
}

type Store struct {
	mu     sync.RWMutex
	byUUID map[string]Player
}

func NewStore() *Store {
	return &Store{byUUID: make(map[string]Player)}
}

func (s *Store) Add(p Player) error {
	if err := p.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.byUUID[p.UUID] = p
	return nil
}

func (s *Store) Get(uuid string) (Player, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.byUUID[uuid]
	return p, ok
}

func (s *Store) Remove(uuid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.byUUID[uuid]; !ok {
		return false
	}

	delete(s.byUUID, uuid)
	return true
}

func (s *Store) List() []Player {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Player, 0, len(s.byUUID))

	for _, p := range s.byUUID {
		out = append(out, p)
	}

	return out
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.byUUID)
}

func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byUUID = make(map[string]Player)
}

var Default = NewStore()

func Add(p Player) error {
	return Default.Add(p)
}

func Get(uuid string) (Player, bool) {
	return Default.Get(uuid)
}

func Remove(uuid string) bool {
	return Default.Remove(uuid)
}

func List() []Player {
	return Default.List()
}

func Count() int {
	return Default.Count()
}

type Executor interface {
	Kick(uuid, reason string) bool
	SendMessage(uuid, text string) bool
}

var (
	execMu sync.RWMutex
	exec   Executor
)

func SetExecutor(e Executor) {
	execMu.Lock()
	defer execMu.Unlock()
	exec = e
}

func currentExecutor() Executor {
	execMu.RLock()
	defer execMu.RUnlock()
	return exec
}

func Kick(uuid, reason string) bool {
	if uuid == "" || len(reason) > MaxReasonLen {
		return false
	}
	if e := currentExecutor(); e != nil {
		return e.Kick(uuid, reason)
	}
	return false
}

// SendMessage unimplemented
func SendMessage(uuid, text string) bool {
	if uuid == "" {
		return false
	}
	if e := currentExecutor(); e != nil {
		return e.SendMessage(uuid, text)
	}
	return false
}
