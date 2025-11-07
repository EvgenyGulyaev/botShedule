package tgpi

import (
	"encoding/json"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Lesson struct {
	Time    int
	Name    string
	Place   string
	Teacher string
	Type    uint8
}

type Schedule struct {
	Day     string
	Lessons []Lesson
}

type rec struct {
	Subject string `json:"subject"`
	Aud     string `json:"aud"`
	Type    uint8  `json:"type"`
	Teacher []struct {
		Name string `json:"teacher"`
	} `json:"teacher"`
	Lesson []int `json:"lesson"`
}

type preload struct {
	Schedule struct {
		Day []struct {
			Date string `json:"date"`
			Rec  []rec  `json:"rec"`
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

	for _, day := range s.Schedule.Day {
		date := day.Date
		lessons := []Lesson{}
		for _, lesson := range day.Rec {
			teachers := make([]string, len(lesson.Teacher))
			for i, t := range lesson.Teacher {
				teachers[i] = t.Name
			}
			teacher := strings.Join(teachers, ",")

			for _, l := range lesson.Lesson {
				lessons = append(lessons, Lesson{
					Time:    l,
					Name:    lesson.Subject,
					Place:   lesson.Aud,
					Teacher: teacher,
					Type:    lesson.Type,
				})
			}

		}
		result = append(result, Schedule{Day: date, Lessons: lessons})
	}

	return
}
