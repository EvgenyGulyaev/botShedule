package tgpi

import (
	"encoding/json"
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

type Lesson struct {
	Time    string
	Name    string
	Place   string
	Teacher string
}

type Schedule struct {
	Day     string
	Lessons []Lesson
}

type rec struct {
	Subject string `json:"subject"`
	Aud     string `json:"aud"`
	Type    int    `json:"type"`
	Teacher []struct {
		Name string `json:"teacher"`
	} `json:"teacher"`
	Lesson []int `json:"lesson"`
}

type preload struct {
	Schedule struct {
		Day []struct {
			Rec []rec `json:"rec"`
		} `json:"day"`
	} `json:"schedule"`
}

func getSchedule(doc *goquery.Document) (result []Schedule) {

	d := doc.Find("body script").Eq(0).Text()
	data := d[15 : len(d)-18]
	var s preload
	err := json.Unmarshal([]byte(data), &s)
	if err != nil {
		return
	}

	fmt.Print(s)
	// TODO: Сделать обработку и приведение к типу Schedule

	return
}
