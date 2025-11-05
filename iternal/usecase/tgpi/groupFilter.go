package tgpi

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
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

func getResult(bodyBytes []byte) (elements []El) {
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
