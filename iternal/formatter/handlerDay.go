package formatter

import (
	"fmt"
	"strings"

	"github.com/EvgenyGulyaev/botShedule/iternal/usecase/tgpi"
)

var time = map[int]string{
	1: "1️⃣08:30-10:05",
	2: "2️⃣10:15-11:50",
	3: "3️⃣12:10-13:45",
	4: "4️⃣14:00-15:35",
	5: "5️⃣15:45-17:20",
	6: "6️⃣17:35-19:10",
	7: "7️⃣19:20-20:55",
	8: "8️⃣21:05-22:40",
}

var tl = map[uint8]string{
	1:  "Лек.",
	2:  "Пр.",
	3:  "ФВ",
	4:  "Лаб.",
	5:  "Зач.",
	6:  "Экз.",
	7:  "ВКР",
	8:  "ГЭ",
	9:  "Конс.",
	10: "ЗПД",
}

func (m *HandlerMessage) prepareDay(s *tgpi.Schedule) string {
	lessons := make([]string, len(s.Lessons))
	for i, l := range s.Lessons {
		lessons[i] = fmt.Sprintf("%s %s %s %s %s", time[l.Time], tl[l.Type], l.Name, l.Teacher, l.Place)
	}
	return fmt.Sprintf("📆%s📆\n%s", s.Day, strings.Join(lessons, "\n"))
}
