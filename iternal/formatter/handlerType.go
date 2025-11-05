package formatter

import "github.com/EvgenyGulyaev/botShedule/iternal/usecase/tgpi"

func handleType(g *tgpi.El) string {
	switch g.Type {
	case tgpi.Aud:
		return "Аудитория"
	case tgpi.Teacher:
		return "Учитель"
	case tgpi.Group:
		return "Группа"
	default:
		return ""
	}
}
