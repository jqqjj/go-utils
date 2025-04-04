package utils

import (
	"math/rand"
	"sync"
)

type Map[K comparable, V any] struct {
	mux sync.RWMutex
	ent map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		ent: make(map[K]V),
	}
}

func (m *Map[K, V]) Load(key K) (val V, loaded bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	val, loaded = m.ent[key]
	return
}

func (m *Map[K, V]) Len() int {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return len(m.ent)
}

func (m *Map[K, V]) WithLock(fn func(m map[K]V)) {
	m.mux.Lock()
	defer m.mux.Unlock()
	fn(m.ent)
}

func (m *Map[K, V]) Filter(fn func(K, V) bool) map[K]V {
	m.mux.RLock()
	defer m.mux.RUnlock()

	result := make(map[K]V)
	for k, v := range m.ent {
		if fn(k, v) {
			result[k] = v
		}
	}
	return result
}

func (m *Map[K, V]) Range(fn func(K, V) bool) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	for k, v := range m.ent {
		if !fn(k, v) {
			return
		}
	}
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (val V, loaded bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if val, ok := m.ent[key]; ok {
		return val, true
	} else {
		m.ent[key] = value
		return value, false
	}
}

func (m *Map[K, V]) Store(key K, value V) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.ent[key] = value
}

func (m *Map[K, V]) Delete(key K) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, ok := m.ent[key]; ok {
		delete(m.ent, key)
		m.tryShrinkLocked()
	}
}

func (m *Map[K, V]) Deletes(keys ...K) {
	m.mux.Lock()
	defer m.mux.Unlock()

	var count int
	for _, key := range keys {
		if _, ok := m.ent[key]; ok {
			delete(m.ent, key)
			count++
		}
	}

	if count > 0 {
		m.tryShrinkLocked()
	}
}

func (m *Map[K, V]) LoadAndDelete(key K) (val V, loaded bool) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if val, ok := m.ent[key]; ok {
		delete(m.ent, key)
		m.tryShrinkLocked()
		return val, true
	}
	var v V
	return v, false
}

func (m *Map[K, V]) Shrink() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.shrinkLocked()
}

func (m *Map[K, V]) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.ent = make(map[K]V)
}

func (m *Map[K, V]) tryShrinkLocked() {
	if rand.Intn(1000) >= 990 {
		m.shrinkLocked()
	}
}

func (m *Map[K, V]) shrinkLocked() {
	newEnt := make(map[K]V)
	for k, v := range m.ent {
		newEnt[k] = v
	}
	m.ent = newEnt
}
