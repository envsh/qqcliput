package main

import (
	"sync"
	"time"
)

type entry struct {
	time time.Time
	text string
}

type textRing struct {
	mu    sync.Mutex
	items []entry
	keep  time.Duration
}

func newTextRing(keep time.Duration) *textRing {
	return &textRing{keep: keep}
}

func (r *textRing) Append(t time.Time, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, entry{t, text})
	r.evict()
}

func (r *textRing) evict() {
	cutoff := time.Now().Add(-r.keep)
	i := 0
	for i < len(r.items) && r.items[i].time.Before(cutoff) {
		i++
	}
	r.items = r.items[i:]
}
