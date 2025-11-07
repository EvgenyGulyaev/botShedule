package formatter

import (
	"fmt"
	"strings"

	"github.com/EvgenyGulyaev/botShedule/iternal/usecase/tgpi"
)

func (m *HandlerMessage) prepareDay(s *tgpi.Schedule) string {
	// Day     string
	// Lessons []Lesson
	lessons := make([]string, len(s.Lessons))
	for i, l := range s.Lessons {
		lessons[i] = fmt.Sprintf("%s %s %s %s %s", getTime(l.Time), getType(l.Type), l.Name, l.Teacher, l.Place)
	}
	return fmt.Sprintf("📆%s📆\n%s", s.Day, strings.Join(lessons, "\n"))
}

func getTime(l int) string {
	switch l {
	case 1:
		return "1️⃣08:30-10:05"
	case 2:
		return "2️⃣10:15-11:50"
	case 3:
		return "3️⃣12:10-13:45"
	case 4:
		return "4️⃣14:00-15:35"
	case 5:
		return "5️⃣15:45-17:20"
	case 6:
		return "6️⃣17:35-19:10"
	case 7:
		return "7️⃣19:20-20:55"
	case 8:
		return "8️⃣21:05-22:40"
	default:
		return ""
	}
}

func getType(t uint8) string {
	switch t {
	case 1:
		return "Лек."
	case 2:
		return "Пр."
	case 3:
		return "ФВ"
	case 4:
		return "Лаб."
	case 5:
		return "Зач."
	case 6:
		return "Экз."
	case 7:
		return "ВКР"
	case 8:
		return "ГЭ"
	case 9:
		return "Конс."
	case 10:
		return "ЗПД"
	default:
		return ""
	}
}
