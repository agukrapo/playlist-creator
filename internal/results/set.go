package results

import (
	"sync"
)

type Set struct {
	list  []string
	table map[string]struct{}
	mu    sync.RWMutex
	empty bool
}

func New(size int) *Set {
	return &Set{
		list:  make([]string, size),
		table: make(map[string]struct{}, size),
	}
}

func (c *Set) Add(i int, value string) bool {
	if value == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.table[value]; ok {
		return false
	}

	if old := c.list[i]; old != "" {
		delete(c.table, old)
	}

	c.list[i] = value
	c.table[value] = struct{}{}

	c.empty = false

	return true
}

func (c *Set) Slice() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]string, 0, len(c.list))
	for _, v := range c.list {
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}

func (c *Set) Empty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.empty
}
