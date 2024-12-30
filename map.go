package utils

import (
	"math/rand"
	"sync"
)

type Map[K comparable, V any] struct {
	sync.RWMutex
	ent map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		ent: make(map[K]V),
	}
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	m.RLock()
	defer m.RUnlock()
	val, ok := m.ent[key]
	return val, ok
}

func (m *Map[K, V]) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.ent)
}

func (m *Map[K, V]) Filter(fn func(K, V) bool) map[K]V {
	m.RLock()
	defer m.RUnlock()

	result := make(map[K]V)
	for k, v := range m.ent {
		if fn(k, v) {
			result[k] = v
		}
	}
	return result
}

func (m *Map[K, V]) Range(fn func(K, V) bool) {
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.ent {
		if !fn(k, v) {
			return
		}
	}
}

func (m *Map[K, V]) Set(key K, value V) {
	m.Lock()
	defer m.Unlock()
	m.ent[key] = value
}

func (m *Map[K, V]) Delete(key K) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.ent[key]; ok {
		delete(m.ent, key)
		m.shrinkLocked()
	}
}

func (m *Map[K, V]) Deletes(keys ...K) {
	m.Lock()
	defer m.Unlock()

	var count int
	for _, key := range keys {
		if _, ok := m.ent[key]; ok {
			delete(m.ent, key)
			count++
		}
	}

	if count > 0 {
		m.shrinkLocked()
	}
}

func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	m.Lock()
	defer m.Unlock()

	if val, ok := m.ent[key]; ok {
		delete(m.ent, key)
		m.shrinkLocked()
		return val, ok
	}
	var v V
	return v, false
}

func (m *Map[K, V]) shrinkLocked() {
	if rand.Intn(100) >= 90 {
		newEnt := make(map[K]V)
		for k, v := range m.ent {
			newEnt[k] = v
		}
		m.ent = newEnt
	}
}
