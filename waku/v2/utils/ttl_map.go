package utils

import (
	"context"
	"sync"
	"time"
)

type elem[V any] struct {
	v          V
	lastAccess int64
}

type TtlMap[K comparable, V any] struct {
	sync.RWMutex

	m map[K]elem[V]
}

func NewTtlMap[K comparable, V any](ctx context.Context, maxTtl uint) *TtlMap[K, V] {
	m := &TtlMap[K, V]{m: make(map[K]elem[V])}
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				func() {
					m.Lock()
					defer m.Unlock()
					now := time.Now().Unix()
					for k, v := range m.m {
						if now-v.lastAccess > int64(maxTtl) {
							delete(m.m, k)
						}
					}
				}()
			}
		}
	}()
	return m
}

func (m *TtlMap[K, V]) Put(k K, v V) {
	m.Lock()
	defer m.Unlock()
	m.m[k] = elem[V]{v, time.Now().Unix()}
}

func (m *TtlMap[K, V]) Get(k K) (V, bool) {
	m.RLock()
	defer m.RUnlock()
	v, ok := m.m[k]
	return v.v, ok
}

func (m *TtlMap[K, V]) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.m)
}
