package websocket_server

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/olahol/melody"
)

type SessionMeta struct {
	ConnID   string
	Guard    string
	UserID   string
	Channels map[string]struct{}
	LastSeen time.Time
}

type Hub struct {
	mu sync.RWMutex

	byConnID map[string]*melody.Session
	meta     map[*melody.Session]*SessionMeta

	byGuard map[string]map[string]struct{}
	byUser  map[string]map[string]map[string]struct{} // guard -> userID -> connID set
	byChan  map[string]map[string]struct{}
}

func NewHub() *Hub {
	return &Hub{
		byConnID: map[string]*melody.Session{},
		meta:     map[*melody.Session]*SessionMeta{},
		byGuard:  map[string]map[string]struct{}{},
		byUser:   map[string]map[string]map[string]struct{}{},
		byChan:   map[string]map[string]struct{}{},
	}
}

func (h *Hub) Attach(s *melody.Session, guard, userID string) *SessionMeta {
	h.mu.Lock()
	defer h.mu.Unlock()

	m := &SessionMeta{
		ConnID:   uuid.NewString(),
		Guard:    guard,
		UserID:   userID,
		Channels: map[string]struct{}{},
		LastSeen: time.Now(),
	}

	h.byConnID[m.ConnID] = s
	h.meta[s] = m

	if _, ok := h.byGuard[guard]; !ok {
		h.byGuard[guard] = map[string]struct{}{}
	}
	h.byGuard[guard][m.ConnID] = struct{}{}

	if _, ok := h.byUser[guard]; !ok {
		h.byUser[guard] = map[string]map[string]struct{}{}
	}
	if _, ok := h.byUser[guard][userID]; !ok {
		h.byUser[guard][userID] = map[string]struct{}{}
	}
	h.byUser[guard][userID][m.ConnID] = struct{}{}

	return m
}

func (h *Hub) Detach(s *melody.Session) {
	h.mu.Lock()
	defer h.mu.Unlock()

	m, ok := h.meta[s]
	if !ok || m == nil {
		return
	}

	delete(h.meta, s)
	delete(h.byConnID, m.ConnID)

	if gset, ok := h.byGuard[m.Guard]; ok {
		delete(gset, m.ConnID)
		if len(gset) == 0 {
			delete(h.byGuard, m.Guard)
		}
	}

	if u1, ok := h.byUser[m.Guard]; ok {
		if uset, ok := u1[m.UserID]; ok {
			delete(uset, m.ConnID)
			if len(uset) == 0 {
				delete(u1, m.UserID)
			}
		}
		if len(u1) == 0 {
			delete(h.byUser, m.Guard)
		}
	}

	for ch := range m.Channels {
		if cset, ok := h.byChan[ch]; ok {
			delete(cset, m.ConnID)
			if len(cset) == 0 {
				delete(h.byChan, ch)
			}
		}
	}
}

func (h *Hub) Touch(s *melody.Session) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.meta[s]; ok && m != nil {
		m.LastSeen = time.Now()
	}
}

func (h *Hub) Subscribe(s *melody.Session, channels []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	m := h.meta[s]
	if m == nil {
		return
	}

	for _, ch := range channels {
		if ch == "" {
			continue
		}
		m.Channels[ch] = struct{}{}
		if _, ok := h.byChan[ch]; !ok {
			h.byChan[ch] = map[string]struct{}{}
		}
		h.byChan[ch][m.ConnID] = struct{}{}
	}
}

func (h *Hub) Unsubscribe(s *melody.Session, channels []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	m := h.meta[s]
	if m == nil {
		return
	}

	for _, ch := range channels {
		if ch == "" {
			continue
		}
		delete(m.Channels, ch)
		if cset, ok := h.byChan[ch]; ok {
			delete(cset, m.ConnID)
			if len(cset) == 0 {
				delete(h.byChan, ch)
			}
		}
	}
}

func (h *Hub) GetMeta(s *melody.Session) *SessionMeta {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.meta[s]
}

func (h *Hub) ListSessions() map[*melody.Session]*SessionMeta {
	h.mu.RLock()
	defer h.mu.RUnlock()

	out := make(map[*melody.Session]*SessionMeta, len(h.meta))
	for s, m := range h.meta {
		out[s] = m
	}
	return out
}

func (h *Hub) Targets(t PushTarget) []*melody.Session {
	h.mu.RLock()
	defer h.mu.RUnlock()

	out := make([]*melody.Session, 0, 8)
	seen := map[*melody.Session]struct{}{}
	add := func(connID string) {
		s, ok := h.byConnID[connID]
		if !ok || s == nil {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}

	if t.ConnID != "" {
		add(t.ConnID)
	}
	for _, id := range t.ConnIDs {
		add(id)
	}

	if t.Channel != "" {
		if cset, ok := h.byChan[t.Channel]; ok {
			for connID := range cset {
				add(connID)
			}
		}
	}

	if t.Guard != "" && t.UserID != "" {
		if g, ok := h.byUser[t.Guard]; ok {
			if u, ok := g[t.UserID]; ok {
				for connID := range u {
					add(connID)
				}
			}
		}
	}

	if t.Guard != "" {
		for _, uid := range t.UserIDs {
			if uid == "" {
				continue
			}
			if g, ok := h.byUser[t.Guard]; ok {
				if u, ok := g[uid]; ok {
					for connID := range u {
						add(connID)
					}
				}
			}
		}
	}

	if t.Broadcast && t.Guard != "" {
		if gset, ok := h.byGuard[t.Guard]; ok {
			for connID := range gset {
				add(connID)
			}
		}
	}

	return out
}
