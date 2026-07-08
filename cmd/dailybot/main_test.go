package main

import (
	"math/rand"
	"testing"
)

func TestRandomWindowOffset(t *testing.T) {
	r := rand.New(rand.NewSource(0))

	t.Run("count less than window returns 0", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			if got := randomWindowOffset(5, 20, r); got != 0 {
				t.Fatalf("expected offset 0 when count < window, got %d", got)
			}
		}
	})

	t.Run("count equal to window returns 0", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			if got := randomWindowOffset(20, 20, r); got != 0 {
				t.Fatalf("expected offset 0 when count == window, got %d", got)
			}
		}
	})

	t.Run("count greater than window stays within bounds", func(t *testing.T) {
		count, window := 4567, 20
		seenNonZero := false
		for i := 0; i < 1000; i++ {
			got := randomWindowOffset(count, window, r)
			if got < 0 || got > count-window {
				t.Fatalf("expected offset in [0, %d], got %d", count-window, got)
			}
			if got != 0 {
				seenNonZero = true
			}
		}
		if !seenNonZero {
			t.Error("expected at least one non-zero offset in 1000 draws")
		}
	})

	t.Run("count zero returns 0", func(t *testing.T) {
		if got := randomWindowOffset(0, 20, r); got != 0 {
			t.Fatalf("expected offset 0 when count is 0, got %d", got)
		}
	})
}
