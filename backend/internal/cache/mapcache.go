// backend/internal/cache/mapcache.go
package cache

import "sync"

type MapCache[K comparable, V any] struct{ m sync.Map }

func NewMapCache[K comparable, V any]() *MapCache[K, V] {
	return &MapCache[K, V]{}
}
func (c *MapCache[K, V]) Set(k K, v V) {
	c.m.Store(k, v)
}
func (c *MapCache[K, V]) Get(k K) (V, bool) {
	v, ok := c.m.Load(k)
	if !ok {
		var z V
		return z, false
	}
	return v.(V), true
}
func (c *MapCache[K, V]) Delete(k K) { c.m.Delete(k) }
func (c *MapCache[K, V]) Clear()     { c.m.Range(func(k, _ any) bool { c.m.Delete(k); return true }) }
