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

// saturdayAllowedSubjectIDs holds the subject IDs permitted on Saturday without
// penalty: NSTP 1/2 (which are Saturday-pinned, taught to 1st year in the 1st
// and 2nd semesters) plus any subject explicitly flagged SaturdayOnly. It is
// installed by RunGeneticAlgorithm before a run via SetSaturdayAllowedSubjects.
// While nil (e.g. outside a GA run) the Saturday-reservation penalty in the
// fitness function is skipped, so other callers are unaffected.
var saturdayAllowedSubjectIDs map[uint16]bool

// SetSaturdayAllowedSubjects builds and installs the package-wide set of subject
// IDs allowed on Saturday so the fitness function can penalise other (regular)
// subjects placed on Saturday and keep it reserved mainly for the 1st-year NSTP.
func SetSaturdayAllowedSubjects(curriculums []Curriculum.Curriculum) {
	allowed := buildNSTP1Or2SubjectIDSet(curriculums)

	for _, curriculum := range curriculums {
		for _, yearLevel := range curriculum.YearLevels {
			for _, semester := range yearLevel.Semesters {
				for _, subject := range semester.Subjects {
					if subject.SaturdayOnly {
						allowed[subject.ID] = true
					}
				}
			}
		}
	}

	saturdayAllowedSubjectIDs = allowed
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
