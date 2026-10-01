package scan

import (
	"container/heap"
	"sort"
	"sync"
)

type fileHeap []File

func (h fileHeap) Len() int           { return len(h) }
func (h fileHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h fileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *fileHeap) Push(x any)        { *h = append(*h, x.(File)) }
func (h *fileHeap) Pop() any {
	old := *h
	f := old[len(old)-1]
	*h = old[:len(old)-1]
	return f
}

// topFiles keeps the N biggest files seen (min-heap).
type topFiles struct {
	mu   sync.Mutex
	keep int
	h    fileHeap
}

func (t *topFiles) offer(f File) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.h) < t.keep {
		heap.Push(&t.h, f)
		return
	}
	if f.Size > t.h[0].Size {
		t.h[0] = f
		heap.Fix(&t.h, 0)
	}
}

func (t *topFiles) sorted() []File {
	t.mu.Lock()
	out := append([]File(nil), t.h...)
	t.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Size > out[j].Size })
	return out
}

// inodeSet remembers hard-linked inodes so each is counted once.
type inodeSet struct {
	shards [64]struct {
		sync.Mutex
		m map[[2]uint64]struct{}
	}
}

// firstSeen reports whether (dev, ino) is new, and records it.
func (s *inodeSet) firstSeen(dev, ino uint64) bool {
	sh := &s.shards[ino%64]
	sh.Lock()
	defer sh.Unlock()
	if sh.m == nil {
		sh.m = make(map[[2]uint64]struct{})
	}
	k := [2]uint64{dev, ino}
	if _, ok := sh.m[k]; ok {
		return false
	}
	sh.m[k] = struct{}{}
	return true
}
