package cache

import "sync"

type BlockUser struct {
	data map[int64]bool
	mu   sync.RWMutex
}

func InitBlockUser() *BlockUser {
	return &BlockUser{data: make(map[int64]bool)}
}

func (b *BlockUser) Block(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.data[chatID] {
		delete(b.data, chatID)
	} else {
		b.data[chatID] = true
	}
}

func (b *BlockUser) IsBlock(chatID int64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data[chatID]
}
