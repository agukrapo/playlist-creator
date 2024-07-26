package results

import (
	"sync"
)

type Set struct {
	list  []string
	table map[string]int
	mu    sync.RWMutex
	empty bool
}

func New(size int) *Set {
	return &Set{
		list:  make([]string, size),
		table: make(map[string]int, size),
	}
}

func (c *Set) Add(i int, value string) (bool, int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if oldIndex, ok := c.table[value]; ok {
		return false, oldIndex
	}

	if old := c.list[i]; old != "" {
		delete(c.table, old)
	}

	c.list[i] = value
	c.table[value] = i

	c.empty = false

	return true, -1
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
