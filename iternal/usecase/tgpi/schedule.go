package tgpi

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
	"github.com/PuerkitoBio/goquery"
)

const (
	scheduleCacheTTL   = 5 * time.Minute
	scheduleCacheLimit = 1000
)

type scheduleKey struct {
	Type TypeEl
	ID   int
}

type scheduleEntry struct {
	value     []Schedule
	checkedAt time.Time
}

type scheduleCache struct {
	mu      sync.Mutex
	entries map[scheduleKey]scheduleEntry
}

func (c *scheduleCache) get(key scheduleKey, load func() ([]Schedule, error)) []Schedule {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entries == nil {
		c.entries = make(map[scheduleKey]scheduleEntry)
	}
	entry, ok := c.entries[key]
	if ok && time.Since(entry.checkedAt) < scheduleCacheTTL {
		return entry.value
	}

	value, err := load()
	if err != nil {
		if ok {
			entry.checkedAt = time.Now()
			c.entries[key] = entry
			return entry.value
		}
		return nil
	}
	if !ok && len(c.entries) >= scheduleCacheLimit {
		c.evictOldest()
	}
	c.entries[key] = scheduleEntry{value: value, checkedAt: time.Now()}
	return value
}

func (c *scheduleCache) evictOldest() {
	var oldestKey scheduleKey
	var oldestTime time.Time
	for key, entry := range c.entries {
		if oldestTime.IsZero() || entry.checkedAt.Before(oldestTime) {
			oldestKey, oldestTime = key, entry.checkedAt
		}
	}
	delete(c.entries, oldestKey)
}

func InitClientSchedule() *Client {
	return singleton.GetInstance("client-schedule", func() interface{} {
		return &Client{
			client: &http.Client{Timeout: 15 * time.Second},
			url:    "https://edu.tgpi.ru/schedule/",
		}
	}).(*Client)
}

func (t *Client) GetSchedule(el *El) []Schedule {
	key := scheduleKey{Type: el.Type, ID: el.ID}
	return t.schedules.get(key, func() ([]Schedule, error) {
		return t.fetchSchedule(el)
	})
}

func (t *Client) fetchSchedule(el *El) ([]Schedule, error) {
	req, err := t.getReqSchedule(el)
	if err != nil {
		return nil, err
	}

	res, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("TGPI schedule: unexpected HTTP status %s", res.Status)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}
	return getSchedule(doc)
}

func (t *Client) getReqSchedule(el *El) (req *http.Request, err error) {
	req, err = http.NewRequest(http.MethodGet, t.getUrlSchedule(el.Type, el.ID), nil)
	if err != nil {
		return
	}
	return
}

func (c *Client) getUrlSchedule(t TypeEl, id int) string {
	return fmt.Sprintf("%s%s/%d/", c.url, t, id)
}
