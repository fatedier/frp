package group

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreate_New(t *testing.T) {
	r := newGroupRegistry[*int]()
	called := 0
	v := 42
	got := r.getOrCreate("k", func() *int { called++; return &v })
	assert.Equal(t, 1, called)
	assert.Equal(t, &v, got)
}

func TestGetOrCreate_Existing(t *testing.T) {
	r := newGroupRegistry[*int]()
	v := 42
	r.getOrCreate("k", func() *int { return &v })

	called := 0
	got := r.getOrCreate("k", func() *int { called++; return nil })
	assert.Equal(t, 0, called)
	assert.Equal(t, &v, got)
}

func TestGet_ExistingAndMissing(t *testing.T) {
	r := newGroupRegistry[*int]()
	v := 1
	r.getOrCreate("k", func() *int { return &v })

	got, ok := r.get("k")
	assert.True(t, ok)
	assert.Equal(t, &v, got)

	_, ok = r.get("missing")
	assert.False(t, ok)
}

func TestIsCurrent(t *testing.T) {
	r := newGroupRegistry[*int]()
	v1 := 1
	v2 := 2
	r.getOrCreate("k", func() *int { return &v1 })

	assert.True(t, r.isCurrent("k", func(g *int) bool { return g == &v1 }))
	assert.False(t, r.isCurrent("k", func(g *int) bool { return g == &v2 }))
	assert.False(t, r.isCurrent("missing", func(g *int) bool { return true }))
}

func TestRemoveIf(t *testing.T) {
	t.Run("removes when fn returns true", func(t *testing.T) {
		r := newGroupRegistry[*int]()
		v := 1
		r.getOrCreate("k", func() *int { return &v })
		r.removeIf("k", func(g *int) bool { return g == &v })
		_, ok := r.get("k")
		assert.False(t, ok)
	})

	t.Run("keeps when fn returns false", func(t *testing.T) {
		r := newGroupRegistry[*int]()
		v := 1
		r.getOrCreate("k", func() *int { return &v })
		r.removeIf("k", func(g *int) bool { return false })
		_, ok := r.get("k")
		assert.True(t, ok)
	})

	t.Run("noop on missing key", func(t *testing.T) {
		r := newGroupRegistry[*int]()
		r.removeIf("missing", func(g *int) bool { return true }) // should not panic
	})
}

func TestConcurrentGetOrCreateAndRemoveIf(t *testing.T) {
	r := newGroupRegistry[*int]()
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := range n {
		v := i
		go func() {
			defer wg.Done()
			r.getOrCreate("k", func() *int { return &v })
		}()
		go func() {
			defer wg.Done()
			r.removeIf("k", func(*int) bool { return true })
		}()
	}
	wg.Wait()

	// After all goroutines finish, accessing the key must not panic.
	require.NotPanics(t, func() {
		_, _ = r.get("k")
	})
}
