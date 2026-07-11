package tgpi

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TypeEl string

type El struct {
	ID   int    `json:"id"`
	Name string `json:"title"`
	Type TypeEl
}

type ElTeacher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type elementGroup struct {
	Aud     []El        `json:"aud"`
	Teacher []ElTeacher `json:"teacher"`
	Group   []El        `json:"group"`
}

const (
	Teacher TypeEl = "teacher"
	Group   TypeEl = "group"
	Aud     TypeEl = "aud"
)

func filterGroups(groupName string, els []El) []El {
	if groupName == "" {
		return els
	}

	mask := strings.ToLower(groupName)
	results := make([]El, 0)
	for _, el := range els {
		if strings.Contains(strings.ToLower(el.Name), mask) {
			results = append(results, el)
		}
	}
	return results
}

func setType(els *[]El, gt TypeEl) {
	for i := range *els {
		(*els)[i].Type = gt
	}
}

func convert(t *[]ElTeacher) (elements []El) {
	for _, v := range *t {
		elements = append(elements, El{ID: v.ID, Name: v.Name, Type: Teacher})
	}
	return
}

func getGroups(bodyBytes []byte) (elements []El) {
	var elGroup elementGroup
	err := json.Unmarshal(bodyBytes, &elGroup)
	if err != nil {
		fmt.Print(err)
		return
	}
	teachers := convert(&elGroup.Teacher)
	setType(&elGroup.Group, Group)
	setType(&elGroup.Aud, Aud)
	return append(elGroup.Group, append(elGroup.Aud, teachers...)...)
}
