package results

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet_Put(t *testing.T) {
	t.Run("same index, different values", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 0, "_B", true))
		requirements(t, true, -1, put(s, 0, "_C", true))

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_C"}, active)
		assert.Empty(t, inactive)
	})
	t.Run("different index, same values", func(t *testing.T) {
		s := New(3)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, false, 0, put(s, 1, "_A", true))
		requirements(t, false, 0, put(s, 2, "_A", true))

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_A"}, active)
		assert.Equal(t, []string{"query_A", "query_A"}, inactive)
	})
	t.Run("different index, different values", func(t *testing.T) {
		s := New(3)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 1, "_B", true))
		requirements(t, true, -1, put(s, 2, "_C", true))

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_A", "id_B", "id_C"}, active)
		assert.Empty(t, inactive)
	})
	t.Run("missing middle index", func(t *testing.T) {
		s := New(4)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 2, "_C", true))

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_A", "id_C"}, active)
		assert.Empty(t, inactive)
	})
	t.Run("should purge old values", func(t *testing.T) {
		s := New(4)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 0, "_B", true))
		requirements(t, true, -1, put(s, 0, "_A", true))

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_A"}, active)
		assert.Empty(t, inactive)
	})
	t.Run("same id and value", func(t *testing.T) {
		s := New(4)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 0, "_A", true))

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_A"}, active)
		assert.Empty(t, inactive)
	})
	t.Run("inactive value", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, put(s, 0, "_A", false))

		assert.True(t, s.Empty())

		active, inactive := s.Slice()
		assert.Empty(t, active)
		assert.Equal(t, []string{"query_A"}, inactive)
	})
	t.Run("active then inactive values", func(t *testing.T) {
		s := New(3)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 1, "_B", true))
		requirements(t, true, -1, put(s, 2, "_C", true))

		assert.False(t, s.Empty())

		active, inactive := s.Slice()
		assert.Equal(t, []string{"id_A", "id_B", "id_C"}, active)
		assert.Empty(t, inactive)

		requirements(t, true, -1, put(s, 0, "_A", false))
		requirements(t, true, -1, put(s, 1, "_B", false))
		requirements(t, true, -1, put(s, 2, "_C", false))

		assert.True(t, s.Empty())

		active, inactive = s.Slice()
		assert.Empty(t, active)
		assert.Equal(t, []string{"query_A", "query_B", "query_C"}, inactive)
	})
	t.Run("active twice then inactive once", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, put(s, 0, "_A", true))
		requirements(t, true, -1, put(s, 0, "_A", true))

		requirements(t, true, -1, put(s, 0, "_A", false))

		assert.True(t, s.Empty())

		active, inactive := s.Slice()
		assert.Empty(t, active)
		assert.Equal(t, []string{"query_A"}, inactive)
	})
	t.Run("inactive twice", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, put(s, 0, "_A", false))
		requirements(t, true, -1, put(s, 0, "_A", false))

		assert.True(t, s.Empty())

		active, inactive := s.Slice()
		assert.Empty(t, active)
		assert.Equal(t, []string{"query_A"}, inactive)
	})
	t.Run("only query AKA not found", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, putItem(s, 0, ParseItem("_query")))

		assert.True(t, s.Empty())

		active, inactive := s.Slice()
		assert.Empty(t, active)
		assert.Equal(t, []string{"_query"}, inactive)

		queries := s.Queries()
		assert.Equal(t, []string{"_query"}, queries)
	})
	t.Run("active item", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, putItem(s, 0, Item{
			query:  "_query",
			id:     "_id",
			name:   "_name",
			active: true,
		}))

		assert.False(t, s.Empty())

		active, inactive := s.Slice()
		assert.Equal(t, []string{"_id"}, active)
		assert.Empty(t, inactive)

		queries := s.Queries()
		assert.Equal(t, []string{">>LOCKED§_id§_name§_query"}, queries)
	})
	t.Run("inactive item", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, putItem(s, 0, Item{
			query:  "_query",
			id:     "_id",
			name:   "_name",
			active: false,
		}))

		assert.True(t, s.Empty())

		active, inactive := s.Slice()
		assert.Empty(t, active)
		assert.Equal(t, []string{"_query"}, inactive)

		queries := s.Queries()
		assert.Equal(t, []string{"_query"}, queries)
	})
	t.Run("different queries same result", func(t *testing.T) {
		s := New(2)

		item := Item{
			query:  "_query",
			id:     "_id",
			name:   "_name",
			active: true,
		}

		requirements(t, true, -1, putItem(s, 0, item))
		requirements(t, false, 0, putItem(s, 1, item.WithQuery("_query2")))

		assert.False(t, s.Empty())

		active, inactive := s.Slice()
		assert.Equal(t, []string{"_id"}, active)
		assert.Equal(t, []string{"_query2"}, inactive)

		queries := s.Queries()
		assert.Equal(t, []string{">>LOCKED§_id§_name§_query", "_query2"}, queries)
	})
	t.Run("different queries same result, select another, then the same", func(t *testing.T) {
		s := New(2)

		base := Item{
			id:     "_id",
			name:   "_name",
			active: false,
		}

		item1 := base.WithQuery("_query1")
		item2 := base.WithQuery("_query2")

		requirements(t, true, -1, putItem(s, 0, item1))
		requirements(t, false, 0, putItem(s, 1, item2))

		requirements(t, true, -1, putItem(s, 0, item1.WithActive(true)))

		item2 = item2.WithID("_id2").WithName("_name2").WithActive(true)
		requirements(t, true, -1, putItem(s, 1, item2))

		item2 = item2.WithActive(false)
		requirements(t, true, -1, putItem(s, 1, item2))

		item2 = item2.WithID("_id").WithName("_name").WithActive(true)
		requirements(t, false, 0, putItem(s, 1, item2))
	})
}

func put(s *Set, i int, v string, a bool) func() (bool, int) {
	return func() (bool, int) {
		item := Item{
			query:  "query" + v,
			id:     "id" + v,
			name:   "name" + v,
			active: a,
		}
		return s.Put(i, item)
	}
}

func putItem(s *Set, i int, item Item) func() (bool, int) {
	return func() (bool, int) {
		return s.Put(i, item)
	}
}

func requirements(t *testing.T, expectedOK bool, expectedI int, target func() (bool, int)) {
	t.Helper()

	actualOK, actualI := target()
	require.Equal(t, expectedOK, actualOK)
	require.Equal(t, expectedI, actualI)
}
