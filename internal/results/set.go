package results

import (
	"sync"
)

type item struct {
	value  string
	active bool
}

type Set struct {
	list  []item
	table map[string]int
	mu    sync.RWMutex
	count uint
}

func New(size int) *Set {
	return &Set{
		list:  make([]item, size),
		table: make(map[string]int, size),
	}
}

func (c *Set) Put(i int, value string, active bool) (bool, int) {
	if exists, idx := c.exists(value); exists && i != idx {
		return false, idx
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	old := c.list[i]

	if old.value != "" {
		delete(c.table, old.value)
	}

	if active && !old.active {
		c.count++
	} else if !active && old.active {
		c.count--
	}

	c.list[i] = item{value, active}
	c.table[value] = i

	return true, -1
}

func (c *Set) exists(value string) (bool, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	index, ok := c.table[value]
	return ok, index
}

func (c *Set) Slice() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]string, 0, len(c.list))
	for _, v := range c.list {
		if v.active && v.value != "" {
			out = append(out, v.value)
		}
	}

	return out
}

func (c *Set) Empty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.count == 0
}
