package vk

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/SevereCloud/vksdk/v3/api"
)

const (
	userNameCacheTTL   = time.Hour
	userNameCacheLimit = 1000
)

type userNameEntry struct {
	value     string
	checkedAt time.Time
}

type userNameCache struct {
	mu      sync.Mutex
	entries map[int]userNameEntry
}

func (c *userNameCache) get(id int, load func() (string, error)) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[int]userNameEntry)
	}
	entry, ok := c.entries[id]
	if ok && time.Since(entry.checkedAt) < userNameCacheTTL {
		return entry.value
	}

	value, err := load()
	if err != nil {
		if ok {
			entry.checkedAt = time.Now()
			c.entries[id] = entry
			return entry.value
		}
		return ""
	}
	if !ok && len(c.entries) >= userNameCacheLimit {
		c.evictOldest()
	}
	c.entries[id] = userNameEntry{value: value, checkedAt: time.Now()}
	return value
}

func (c *userNameCache) evictOldest() {
	oldestID := 0
	var oldestTime time.Time
	for id, entry := range c.entries {
		if oldestTime.IsZero() || entry.checkedAt.Before(oldestTime) {
			oldestID, oldestTime = id, entry.checkedAt
		}
	}
	delete(c.entries, oldestID)
}

func (b *Bot) getUserName(id int) string {
	return b.users.get(id, func() (string, error) {
		users, err := b.api.UsersGet(api.Params{
			"user_ids": []int{id},
			"fields":   "first_name,last_name",
		})
		if err != nil {
			return "", err
		}
		if len(users) == 0 {
			return "", errors.New("VK user not found")
		}
		return fmt.Sprintf("%s %s", users[0].FirstName, users[0].LastName), nil
	})
}
