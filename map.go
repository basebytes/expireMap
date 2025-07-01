package expireMap

import (
	"context"
	"sync"
	"time"
)

func New[T any](size int, expire time.Duration) *Map[T] {
	return (&Map[T]{expire: expire}).init(size)
}

type Map[T any] struct {
	values  map[string]T
	cancels map[string]context.CancelFunc
	expire  time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
	lock    sync.RWMutex
}

func (m *Map[T]) init(size int) *Map[T] {
	m.values = make(map[string]T, size)
	m.cancels = make(map[string]context.CancelFunc, size)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

func (m *Map[T]) Set(key string, value T) (T, bool) {
	return m.SetWithExpire(key, value, m.expire)
}

func (m *Map[T]) Get(key string) (v T, b bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	v, b = m.values[key]
	return
}

func (m *Map[T]) Del(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if cancel, ok := m.cancels[key]; ok {
		delete(m.cancels, key)
		delete(m.values, key)
		cancel()
	}
}

func (m *Map[T]) Exist(key string) (b bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, b = m.values[key]
	return
}

func (m *Map[T]) Size() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.values)
}

func (m *Map[T]) SetWithExpire(key string, value T, expire time.Duration) (T, bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if v, ok := m.values[key]; ok {
		return v, false
	}
	ctx, cancel := context.WithTimeout(m.ctx, expire)
	m.cancels[key] = cancel
	m.values[key] = value
	go func() {
		<-ctx.Done()
		m.Del(key)
	}()
	return value, true
}
