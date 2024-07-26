package results

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet_Add(t *testing.T) {
	t.Run("same index, different ids", func(t *testing.T) {
		s := New(1)

		requirements(t, true, -1, testAdd(s, 0, "_A"))
		requirements(t, true, -1, testAdd(s, 0, "_B"))
		requirements(t, true, -1, testAdd(s, 0, "_C"))

		assert.Equal(t, []string{"_C"}, s.Slice())
	})
	t.Run("different index, same ids", func(t *testing.T) {
		s := New(3)

		requirements(t, true, -1, testAdd(s, 0, "_A"))
		requirements(t, false, 0, testAdd(s, 1, "_A"))
		requirements(t, false, 0, testAdd(s, 2, "_A"))

		assert.Equal(t, []string{"_A"}, s.Slice())
	})
	t.Run("different index, different ids", func(t *testing.T) {
		s := New(3)

		requirements(t, true, -1, testAdd(s, 0, "_A"))
		requirements(t, true, -1, testAdd(s, 1, "_B"))
		requirements(t, true, -1, testAdd(s, 2, "_C"))

		assert.Equal(t, []string{"_A", "_B", "_C"}, s.Slice())
	})
	t.Run("missing middle index", func(t *testing.T) {
		s := New(4)

		requirements(t, true, -1, testAdd(s, 0, "_A"))
		requirements(t, true, -1, testAdd(s, 2, "_C"))

		assert.Equal(t, []string{"_A", "_C"}, s.Slice())
	})
	t.Run("should purge old values", func(t *testing.T) {
		s := New(4)

		requirements(t, true, -1, testAdd(s, 0, "_A"))
		requirements(t, true, -1, testAdd(s, 0, "_B"))
		requirements(t, true, -1, testAdd(s, 0, "_A"))

		assert.Equal(t, []string{"_A"}, s.Slice())
	})
}

func testAdd(s *Set, i int, v string) func() (bool, int) {
	return func() (bool, int) {
		return s.Add(i, v)
	}
}

func requirements(t *testing.T, expectedOK bool, expectedI int, values func() (bool, int)) {
	t.Helper()

	actualOK, actualI := values()
	require.Equal(t, expectedOK, actualOK)
	require.Equal(t, expectedI, actualI)
}
