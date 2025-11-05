package formatter

import (
	"fmt"
	"strings"

	"github.com/EvgenyGulyaev/botShedule/iternal/usecase/tgpi"
)

type HandlerMessage struct {
	UserName string
	From     string
	ChatId   int64
}

func (m *HandlerMessage) GetAnswer() (int64, string) {
	tgpi := tgpi.InitClientGroup()
	g := tgpi.GetGroups(m.From)

	return m.ChatId, m.prepareGroups(&g)
}

func (m *HandlerMessage) prepareGroups(gs *[]tgpi.El) string {
	if len(*gs) == 1 {
		return m.prepareSchedule(&(*gs)[0])
	}

	names := make([]string, len(*gs))
	for i, g := range *gs {
		names[i] = fmt.Sprintf("%s: %s", handleType(&g), g.Name)
	}
	return strings.Join(names, ", \n") + "\n"
}

func (m *HandlerMessage) prepareSchedule(g *tgpi.El) string {
	return fmt.Sprintf("Расписание для группы: %s", g.Name)
}
