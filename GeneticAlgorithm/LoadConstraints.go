package GeneticAlgorithm

import (
	"fmt"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

// These two validators enforce the unit cap and the 4-day packing rule as HARD
// constraints WITHIN the genetic algorithm only. They are intentionally NOT
// folded into HorizontalValidation, which is also called by ordinary read/serve
// routes — doing so would retroactively flag pre-existing schedules generated
// under the previous (6-day, uncapped) rules as invalid. Instead these run at
// the same GA gates as HorizontalValidation (after mutation, on crossover
// offspring, and on the final fittest individual), so the GA's existing
// revert/clone machinery keeps the whole population compliant.

// BuildSubjectUnitsMap maps every curriculum subject ID to its credit units.
func BuildSubjectUnitsMap(curriculums []Curriculum.Curriculum) map[uint16]uint8 {
	subject_id_to_units := make(map[uint16]uint8)
	for _, curriculum := range curriculums {
		for _, year_level := range curriculum.YearLevels {
			for _, semester := range year_level.Semesters {
				for _, subject := range semester.Subjects {
					subject_id_to_units[subject.ID] = subject.Units
				}
			}
		}
	}
	return subject_id_to_units
}

// ValidateFourDayPacking returns one error per section whose non-NSTP classes
// are spread across more than Const.MAX_NON_NSTP_SCHOOL_DAYS distinct days.
// NSTP 1/2 subjects are Saturday-pinned and excluded from the day count.
func ValidateFourDayPacking(
	university_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	department_to_validate map[uint16]bool,
	selected_semester int,
) []error {
	if !ENFORCE_FOUR_DAY_PACKING {
		return nil
	}

	nstp_subject_ids := buildNSTP1Or2SubjectIDSet(curriculums)
	errs := make([]error, 0)

	IterateSectionsWeekSchedule(university_sched, curriculums, selected_semester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			if values.WeekSched == nil || values.Curriculum == nil {
				return IterProceed
			}

			if department_to_validate != nil && !department_to_validate[values.Curriculum.DepartmentID] {
				return IterProceed
			}

			used_non_nstp_days := make(map[int]bool)
			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
					subject_id := university_sched[indicies.Usi][day][time_slot].GetSubjectID()
					if subject_id == 0 || nstp_subject_ids[subject_id] {
						continue
					}
					used_non_nstp_days[day] = true
				}
			}

			if len(used_non_nstp_days) > Const.MAX_NON_NSTP_SCHOOL_DAYS {
				errs = append(errs, fmt.Errorf(
					"section %s %s %s (usi:%d) spreads non-NSTP classes across %d days, exceeding the %d-day limit",
					values.Curriculum.CurriculumCode, values.Semester.Name, Curriculum.SECTION[indicies.Section],
					indicies.Usi, len(used_non_nstp_days), Const.MAX_NON_NSTP_SCHOOL_DAYS,
				))
			}

			return IterProceed
		},
	)

	return errs
}

// ValidateInstructorUnitLoad returns one error per instructor whose total
// assigned credit units across the WHOLE university schedule exceed their
// effective weekly unit cap. Computed university-wide (department filter is
// intentionally ignored) because an instructor's cap covers their entire load,
// including classes in other departments. Subjects with 0 units do not count.
func ValidateInstructorUnitLoad(
	university_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	instructor_id_to_instructor map[uint16]*Instructors.Instructor,
	selected_semester int,
) []error {
	subject_id_to_units := BuildSubjectUnitsMap(curriculums)

	// instructor id -> total assigned units (each subject in a section counted once).
	instructor_units := make(map[uint16]uint16)
	// guard so a subject taught across multiple time-slot blocks in a section
	// (e.g. a lec+lab double block) is only counted once.
	type sectionSubjectKey struct {
		usi       int
		subjectID uint16
	}
	counted := make(map[sectionSubjectKey]bool)

	IterateSectionsWeekSchedule(university_sched, curriculums, selected_semester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			if values.WeekSched == nil {
				return IterProceed
			}

			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
					subject_id := university_sched[indicies.Usi][day][time_slot].GetSubjectID()
					if subject_id == 0 {
						continue
					}

					key := sectionSubjectKey{usi: indicies.Usi, subjectID: subject_id}
					if counted[key] {
						continue
					}
					counted[key] = true

					instructor_id := university_sched[indicies.Usi][day][time_slot].GetInstructorID()
					instructor_units[instructor_id] += uint16(subject_id_to_units[subject_id])
				}
			}

			return IterProceed
		},
	)

	errs := make([]error, 0)
	for instructor_id, total_units := range instructor_units {
		max_units := uint16(Const.REGULAR_INSTRUCTOR_MAX_UNITS)
		if instructor, has := instructor_id_to_instructor[instructor_id]; has && instructor != nil {
			max_units = uint16(instructor.EffectiveMaxUnits())
		}

		if total_units > max_units {
			errs = append(errs, fmt.Errorf(
				"instructor id %d is overloaded with %d units, exceeding their cap of %d units",
				instructor_id, total_units, max_units,
			))
		}
	}

	return errs
}
