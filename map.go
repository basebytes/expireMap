package expireMap

import (
	"sync"
	"time"
)

func New[T any](size int, expire time.Duration) *Map[T] {
	return &Map[T]{
		values: make(map[string]*ele[T], size),
		expire: expire,
	}
}

type Map[T any] struct {
	values map[string]*ele[T]
	expire time.Duration
	lock   sync.RWMutex
}

func (m *Map[T]) Del(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if old, ok := m.values[key]; ok {
		old.stop()
		delete(m.values, key)
	}
}

func (m *Map[T]) Get(key string) (T, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	var v T
	if value, ok := m.values[key]; ok {
		value.refresh()
		return value.v, ok
	}
	return v, false
}

func (m *Map[T]) Set(key string, value T) (T, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if old, ok := m.values[key]; ok {
		old.refresh()
		return old.v, false
	} else {
		e := NewEle(value)
		m.values[key] = e
		go e.run(key, m.expire, m)
		return value, true
	}
}

func (m *Map[T]) Size() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.values)
}

func (m *Map[T]) expired(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.values, key)
}

func NewEle[T any](value T) *ele[T] {
	return &ele[T]{v: value, s: make(chan string, 2)}
}

type ele[T any] struct {
	v T
	s chan string
}

func (e *ele[T]) stop() {
	e.s <- "s"
}

func (e *ele[T]) refresh() {
	e.s <- "r"
}

func (e *ele[T]) run(key string, expire time.Duration, m *Map[T]) {
	timer := time.NewTimer(expire)
	defer timer.Stop()
	for {
		select {
		case s := <-e.s:
			if s == "s" {
				return
			}
		case <-timer.C:
			m.expired(key)
			return
		}
		timer.Reset(expire)
	}
}
