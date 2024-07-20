package set

import (
	"fmt"
	"sync"
)

type Set struct {
	list  []string
	table map[string]string
	mu    sync.RWMutex
}

func New(size int) *Set {
	return &Set{
		list:  make([]string, size),
		table: make(map[string]string, size),
	}
}

func (c *Set) Add(i int, id string, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if old, ok := c.table[id]; ok {
		return fmt.Errorf("id %s duplicated: new %q, old %q", id, value, old)
	}

	c.list[i] = id
	c.table[id] = value

	return nil
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

func (c *Set) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.table)
}
