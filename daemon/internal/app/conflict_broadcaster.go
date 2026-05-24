package app

import (
	"sync"

	"github.com/your-org/cortado/daemon/internal/filesync"
)

type ConflictBroadcaster struct {
	mu          sync.Mutex
	subscribers map[chan filesync.ConflictNotice]struct{}
}

func NewConflictBroadcaster() *ConflictBroadcaster {
	return &ConflictBroadcaster{
		subscribers: make(map[chan filesync.ConflictNotice]struct{}),
	}
}

func (b *ConflictBroadcaster) PublishConflict(notice filesync.ConflictNotice) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for subscriber := range b.subscribers {
		select {
		case subscriber <- notice:
		default:
		}
	}
}

func (b *ConflictBroadcaster) Subscribe() (<-chan filesync.ConflictNotice, func()) {
	ch := make(chan filesync.ConflictNotice, 8)

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		if _, ok := b.subscribers[ch]; ok {
			delete(b.subscribers, ch)
			close(ch)
		}
		b.mu.Unlock()
	}
}
