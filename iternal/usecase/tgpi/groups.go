package tgpi

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/EvgenyGulyaev/botShedule/pkg/singleton"
)

const groupsCacheTTL = time.Hour

type Client struct {
	client *http.Client
	url    string

	groupsMu       sync.Mutex
	groups         []El
	groupsLoadedAt time.Time
}

type Params struct {
	Query []string `json:"query"`
	Year  int      `json:"year"`
}

func InitClientGroup() *Client {
	return singleton.GetInstance("client-group", func() interface{} {
		return &Client{
			client: &http.Client{Timeout: 1000 * time.Second},
			url:    "https://edu.tgpi.ru/query/",
		}
	}).(*Client)
}

func (t *Client) GetGroups(groupName string) []El {
	return filterGroups(groupName, t.cachedGroups())
}

func (t *Client) cachedGroups() []El {
	t.groupsMu.Lock()
	defer t.groupsMu.Unlock()

	if !t.groupsLoadedAt.IsZero() && time.Since(t.groupsLoadedAt) < groupsCacheTTL {
		return t.groups
	}

	groups, err := t.fetchGroups()
	if err != nil {
		return t.groups
	}
	t.groups = groups
	t.groupsLoadedAt = time.Now()
	return t.groups
}

func (t *Client) fetchGroups() ([]El, error) {
	req, err := t.getReqGroup()
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("TGPI groups: unexpected HTTP status %s", resp.Status)
	}

	reader, err := getReader(resp)
	if err != nil {
		return nil, err
	}
	if reader != resp.Body {
		defer reader.Close()
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return getGroups(body)
}

func (t *Client) getReqGroup() (req *http.Request, err error) {
	body := Params{
		Query: []string{
			"aud",
			"teacher",
			"group",
		},
		Year: getYear(),
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(body)
	if err != nil {
		return
	}

	req, err = http.NewRequest(http.MethodPost, t.url, &buf)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "ru,en;q=0.9,ru-RU;q=0.8,en-US;q=0.7")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://edu.tgpi.ru")
	req.Header.Set("Referer", "https://edu.tgpi.ru/schedule/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Cookie", "FloaterHiCon=0; FloaterBigFon=0; FloaterSimply=0; _ym_uid=1762025775534324035; _ym_d=1762025775; _ym_isad=1; ASP.NET_SessionId=qjtf0y0io4sdvzp141l1ryth; __AntiXsrfToken=2f3a196970134811af13af493bf4447c")
	req.Header.Set("Accept-Encoding", "identity")

	return
}

func getYear() int {
	year := time.Now().Year()
	if time.Now().Month() < 8 {
		year = year - 1
		return year
	}

	return year
}

func getReader(resp *http.Response) (io.ReadCloser, error) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		return gzip.NewReader(resp.Body)
	}
	return resp.Body, nil
}
