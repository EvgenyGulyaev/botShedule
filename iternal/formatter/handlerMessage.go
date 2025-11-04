package formatter

import (
	"strings"

	usecase "github.com/EvgenyGulyaev/botShedule/iternal/usecase/tgpi"
)

type HandlerMessage struct {
	UserName string
	From     string
	ChatId   int64
}

func (m *HandlerMessage) GetAnswer() (int64, string) {
	tgpi := usecase.InitClient()
	g := tgpi.GetGroups(m.From)

	return m.ChatId, m.prepareGroups(g)
}

func (m *HandlerMessage) prepareGroups(gs []usecase.El) string {
	names := make([]string, len(gs))
	for i, g := range gs {
		names[i] = "Группа: " + g.Name
	}
	return strings.Join(names, ", \n") + "\n"
}
