package no6

import (
	"fmt"
	"sort"
	"sync"
)

type Hashable = comparable

type hashSet[T Hashable] struct {
	m  map[T]struct{}
	mu sync.Mutex
}

func newHashSet[T Hashable]() *hashSet[T] {
	return &hashSet[T]{
		m: make(map[T]struct{}),
	}
}

func (h *hashSet[T]) Add(v ...T) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.add(v...)
}

func (h *hashSet[T]) add(v ...T) {
	for _, i := range v {
		h.m[i] = struct{}{}
	}
}

func (h *hashSet[T]) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.len()
}

func (h *hashSet[T]) len() int {
	return len(h.m)
}

func (h *hashSet[T]) Contains(v T) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.contains(v)
}

func (h *hashSet[T]) contains(v T) bool {
	_, ok := h.m[v]
	return ok
}

func (h *hashSet[T]) AsSlice() []T {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.asSlice()
}

func (h *hashSet[T]) asSlice() []T {
	var s []T
	for k := range h.m {
		s = append(s, k)
	}
	return s
}

func (h *hashSet[T]) AsSortedSlice() []T {
	s := h.AsSlice()
	sort.Slice(s, func(a, b int) bool {
		return fmt.Sprintf("%+v", s[a]) < fmt.Sprintf("%+v", s[b])
	})
	return s
}
