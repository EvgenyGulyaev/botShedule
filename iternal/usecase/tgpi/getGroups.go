package usecase

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	client *http.Client
	url    string
}

type Params struct {
	Query []string `json:"query"`
	Year  int      `json:"year"`
}

type El struct {
	ID   int    `json:"id"`
	Name string `json:"title"`
}

type elementGroup struct {
	Aud     []El `json:"aud"`
	Teacher []El `json:"teacher"`
	Group   []El `json:"group"`
}

func InitClient() *Client {
	return &Client{
		client: &http.Client{Timeout: 1000 * time.Second},
		url:    "https://edu.tgpi.ru/query/",
	}
}

func (t *Client) GetGroups(groupName string) (result []El) {
	req, err := t.getRequest()
	if err != nil {
		return
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	reader := getReader(resp)
	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return
	}
	return filter(groupName, getResult(bodyBytes))
}

func (t *Client) getRequest() (req *http.Request, err error) {
	body := Params{
		Query: []string{
			"aud",
			"teacher",
			"group",
		},
		Year: time.Now().Year(),
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(body)
	if err != nil {
		return
	}

	req, err = http.NewRequest(http.MethodPost, t.url+"query/", &buf)
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

func filterEl(mask string, el El, sem chan struct{}, wg *sync.WaitGroup, mu *sync.Mutex, results *[]El) {
	defer wg.Done()
	defer func() { <-sem }()
	mu.Lock()
	if strings.Contains(el.Name, mask) {
		*results = append(*results, el)
	}
	mu.Unlock()
}

func filter(groupName string, els []El) []El {
	if groupName == "" {
		return els
	}

	var results []El
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 5) // максимум 5 горутин одновременно

	for _, item := range els {
		sem <- struct{}{}
		wg.Add(1)
		go filterEl(groupName, item, sem, &wg, &mu, &results)
	}

	wg.Wait()
	return results
}

func getResult(bodyBytes []byte) (elements []El) {
	var elGroup elementGroup
	err := json.Unmarshal(bodyBytes, &elGroup)
	if err != nil {
		fmt.Print(err)
		return
	}
	return append(elGroup.Group, append(elGroup.Aud, elGroup.Teacher...)...)
}

func getReader(resp *http.Response) (reader io.ReadCloser) {
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			panic(err)
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}
	return
}
