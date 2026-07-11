package tgpi

import (
	"encoding/json"
	"strings"
)

type TypeEl string

type El struct {
	ID   int    `json:"id"`
	Name string `json:"title"`
	Type TypeEl

	searchName string
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
		name := el.searchName
		if name == "" {
			name = strings.ToLower(el.Name)
		}
		if strings.Contains(name, mask) {
			results = append(results, el)
		}
	}
	return results
}

func setType(els *[]El, gt TypeEl) {
	for i := range *els {
		(*els)[i].Type = gt
		(*els)[i].searchName = strings.ToLower((*els)[i].Name)
	}
}

func convert(t *[]ElTeacher) (elements []El) {
	for _, v := range *t {
		elements = append(elements, El{
			ID:         v.ID,
			Name:       v.Name,
			Type:       Teacher,
			searchName: strings.ToLower(v.Name),
		})
	}
	return
}

func getGroups(bodyBytes []byte) ([]El, error) {
	var elGroup elementGroup
	if err := json.Unmarshal(bodyBytes, &elGroup); err != nil {
		return nil, err
	}
	teachers := convert(&elGroup.Teacher)
	setType(&elGroup.Group, Group)
	setType(&elGroup.Aud, Aud)
	return append(elGroup.Group, append(elGroup.Aud, teachers...)...), nil
}
