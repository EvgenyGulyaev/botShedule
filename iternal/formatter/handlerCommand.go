package formatter

import (
	"github.com/EvgenyGulyaev/botShedule/iternal/command"
)

func (m *HandlerMessage) HandlerCommand() string {
	if m.From == "/start" {
		return command.Exec(&command.Start{}, m.UserName)
	}
	return ""
}
