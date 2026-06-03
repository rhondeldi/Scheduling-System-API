package GeneticAlgorithm

import (
	"regexp"
	"strings"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

var nstp1Or2Pattern = regexp.MustCompile(`(?i)\bnstp\s*[-_/]?\s*(1|2)\b`)

func saturdayDayIndex() int {
	return Const.N_WEEKLY_SCHOOL_DAYS - 1
}

func isNSTP1Or2Subject(subject Curriculum.Subject) bool {
	return isNSTP1Or2Text(subject.Code) || isNSTP1Or2Text(subject.Name)
}

func isNSTP1Or2Text(v string) bool {
	value := strings.TrimSpace(v)
	if value == "" {
		return false
	}

	return nstp1Or2Pattern.MatchString(value)
}

func buildNSTP1Or2SubjectIDSet(curriculums []Curriculum.Curriculum) map[uint16]bool {
	nstpSubjectIDs := make(map[uint16]bool)

	for _, curriculum := range curriculums {
		for _, yearLevel := range curriculum.YearLevels {
			if !yearLevel.IsActive {
				continue
			}

			for _, semester := range yearLevel.Semesters {
				for _, subject := range semester.Subjects {
					if !isNSTP1Or2Subject(subject) {
						continue
					}

					nstpSubjectIDs[subject.ID] = true
				}
			}
		}
	}

	return nstpSubjectIDs
}

func weekDayHasNSTP1Or2(
	week *Schedule.WeekTimeTable,
	day int,
	nstpSubjectIDs map[uint16]bool,
) bool {
	if week == nil || day < 0 || day >= Const.N_WEEKLY_SCHOOL_DAYS {
		return false
	}

	for timeSlot := 0; timeSlot < Const.N_DAILY_TIME_SLOTS; timeSlot++ {
		subjectID := week[day][timeSlot].GetSubjectID()
		if subjectID == 0 {
			continue
		}

		if nstpSubjectIDs[subjectID] {
			return true
		}
	}

	return false
}
