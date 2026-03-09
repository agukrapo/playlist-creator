package results

import (
	"fmt"
	"strings"
	"sync"
)

const locked = ">>LOCKED"

type Item struct {
	query    string
	id, name string
	active   bool
}

func ParseItem(in string) Item {
	if chunks := strings.Split(in, "§"); len(chunks) == 4 {
		return Item{
			query:  chunks[3],
			id:     chunks[1],
			name:   chunks[2],
			active: true,
		}
	}

	return Item{query: in}
}

func (i Item) String() string {
	if i.active {
		return fmt.Sprintf("%s§%s§%s§%s", locked, i.id, i.name, i.query)
	}
	return i.query
}

func (i Item) Active() bool {
	return i.active
}

func (i Item) Query() string {
	if i.query == "" {
		panic("empty item query")
	}
	return i.query
}

func (i Item) Name() string {
	if i.name == "" {
		panic("empty item name")
	}
	return i.name
}

func (i Item) WithID(id string) Item {
	return Item{
		query:  i.query,
		id:     id,
		name:   i.name,
		active: i.active,
	}
}

func (i Item) WithName(name string) Item {
	return Item{
		query:  i.query,
		id:     i.id,
		name:   name,
		active: i.active,
	}
}

func (i Item) WithActive(active bool) Item {
	return Item{
		query:  i.query,
		id:     i.id,
		name:   i.name,
		active: active,
	}
}

func (i Item) WithQuery(query string) Item {
	return Item{
		query:  query,
		id:     i.id,
		name:   i.name,
		active: i.active,
	}
}

type Set struct {
	list  []Item
	ids   map[string]int
	mu    sync.RWMutex
	count uint
}

func New(size int) *Set {
	return &Set{
		list: make([]Item, size),
		ids:  make(map[string]int, size),
	}
}

func (c *Set) Put(i int, item Item) (bool, int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	old := c.list[i]

	if item.active && !old.active {
		c.count++
	} else if !item.active && old.active {
		c.count--
	}

	c.list[i] = item

	if item.id != "" {
		if idx, ok := c.ids[item.id]; ok && i != idx {
			c.list[i].active = false
			return false, idx
		}

		c.ids[item.id] = i
	}

	return true, -1
}

func (c *Set) Slice() ([]string, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	active := make([]string, 0, len(c.list))
	inactive := make([]string, 0, len(c.list))
	for _, v := range c.list {
		if v.id != "" && v.active {
			active = append(active, v.id)
		}

		if v.query != "" && !v.active {
			inactive = append(inactive, v.query)
		}
	}

	return active, inactive
}

func (c *Set) Queries() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]string, 0, len(c.list))

	for _, v := range c.list {
		if v.query == "" {
			continue
		}

		out = append(out, v.String())
	}

	return out
}

func (c *Set) Empty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.count == 0
}
