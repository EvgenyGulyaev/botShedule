package tgpi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
	"github.com/PuerkitoBio/goquery"
)

func InitClientSchedule() *Client {
	return singleton.GetInstance("client-schedule", func() interface{} {
		return &Client{
			client: &http.Client{Timeout: 1000 * time.Second},
			url:    "https://edu.tgpi.ru/schedule/",
		}
	}).(*Client)
}

func (t *Client) GetSchedule(el *El) (result []Schedule) {
	req, err := t.getReqSchedule(el)
	if err != nil {
		return
	}

	res, err := t.client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return
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
