package cache

import "sync"

type BlockUser struct {
	data map[int64]bool
	mu   sync.Mutex
}

func InitBlockUser() *BlockUser {
	return &BlockUser{data: make(map[int64]bool)}
}

func (b *BlockUser) Block(chatID int64) {
	_, ok := b.data[chatID]
	b.mu.Lock()
	defer b.mu.Unlock()
	if ok {
		delete(b.data, chatID)
		return
	}
	b.data[chatID] = true
}

func (b *BlockUser) IsBlock(chatID int64) bool {
	_, ok := b.data[chatID]
	return ok
}
