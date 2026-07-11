package formatter

import (
	"fmt"
	"strings"

	"github.com/EvgenyGulyaev/botShedule/iternal/usecase/tgpi"
)

type TypeBot int

const (
	Tg TypeBot = iota
	Vk
)

type HandlerMessage struct {
	UserName string
	From     string
	ChatId   int64
	Type     TypeBot
}

func (m *HandlerMessage) GetAnswer() (string, []string) {
	if command := m.HandlerCommand(); command != "" {
		return command, nil
	}

	tgpi := tgpi.InitClientGroup()
	g := tgpi.GetGroups(m.From)

	if len(g) == 0 {
		return fmt.Sprintf("🎴По вашему запросу: %s Ничего не найдено🎴\n", m.From), nil
	}
	return m.prepareGroups(&g)
}

func (m *HandlerMessage) prepareGroups(gs *[]tgpi.El) (string, []string) {
	if len(*gs) == 1 {
		return m.prepareSchedule(&(*gs)[0]), nil
	}

	names := make([]string, len(*gs))
	for i, g := range *gs {
		names[i] = g.Name
	}
	return "⚠️Выберите из предложенных:⚠️", names
}

func (m *HandlerMessage) prepareSchedule(el *tgpi.El) string {
	tgpi := tgpi.InitClientSchedule()
	gs := tgpi.GetSchedule(el)

	days := make([]string, len(gs))
	for i, g := range gs {
		days[i] = m.prepareDay(&g)
	}

	return fmt.Sprintf("🎴Ваш запрос: %s🎴\n %s", el.Name, strings.Join(days, "\n\n"))
}
