package GeneticAlgorithm

import (
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

type IterReturnType int

const (
	IterProceed             IterReturnType = 0
	IterBreakCurriculumLoop IterReturnType = 1
)

type IterIndices struct {
	Usi        int // university time table schedule index
	Curriculum int // current curriculum index
	YearLevel  int // current year level index
	Semester   int // current semester index
	Section    int // current section index
}

type IterValues struct {
	Sched      Schedule.UniTimeTables  // whole university time table schedule
	WeekSched  *Schedule.WeekTimeTable // current week time table schedule
	Curriculum *Curriculum.Curriculum  // current curriculum
	YearLevel  *Curriculum.YearLevel   // current year level
	Semester   *Curriculum.Semester    // current semester
}

func IsDepartmentScheduleEmpty(
	full_uni_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	selected_semester int,
	department_to_check map[uint16]bool,
) bool {
	is_empty := true

	IterateSectionsWeekSchedule(full_uni_sched, curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		is_to_check, has_key := department_to_check[values.Curriculum.DepartmentID]

		if has_key && is_to_check {
			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
					if values.WeekSched[day][time_slot].GetSubjectID() != 0 {
						is_empty = false
						return IterBreakCurriculumLoop
					}
				}
			}
		}

		return IterProceed
	})

	return is_empty
}

// Iterates over the sections of the week schedule.
//
// ERROR: This function can panic if the input `sched` length is not equal to the total number of sections for the selected semester in the curriculums.
func IterateSectionsWeekSchedule(
	sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	selected_semester int,

	fn_curriculum_block func(indicies IterIndices, values IterValues) IterReturnType,
	fn_semester_block func(indicies IterIndices, values IterValues) IterReturnType,

	fn_section_block func(
		indicies IterIndices, values IterValues,
	) IterReturnType,
) {
	usi := 0

curriculum_loop:
	for curriculum_idx, curriculum := range curriculums {

		if fn_curriculum_block != nil {
			irt_curriculum_block := fn_curriculum_block(
				IterIndices{
					Usi:        usi,
					Curriculum: curriculum_idx,
				},
				IterValues{
					Sched:      sched,
					Curriculum: &curriculum,
				},
			)

			switch irt_curriculum_block {
			case IterBreakCurriculumLoop:
				break curriculum_loop
			}
		}

		for year_level_idx, year_level := range curriculum.YearLevels {

			if !year_level.IsActive {
				continue // skip inactive year levels
			}

			if selected_semester < 0 || selected_semester >= len(year_level.Semesters) {
				continue // skip invalid semester index
			}

			semester := &year_level.Semesters[selected_semester]
			semester_idx := selected_semester

			if fn_semester_block != nil {
				irt_semester_block := fn_semester_block(
					IterIndices{
						Usi:        usi,
						Curriculum: curriculum_idx,
						YearLevel:  year_level_idx,
						Semester:   semester_idx,
					},
					IterValues{
						Sched:      sched,
						Curriculum: &curriculum,
						YearLevel:  &year_level,
						Semester:   semester,
					},
				)

				switch irt_semester_block {
				case IterBreakCurriculumLoop:
					break curriculum_loop
				}
			}

			for section_idx := 0; section_idx < semester.Sections; section_idx++ {

				if fn_section_block != nil {

					var week_time_table *Schedule.WeekTimeTable

					if len(sched) > 0 {
						week_time_table = &sched[usi]
					} else {
						week_time_table = nil
					}

					irt_section_block := fn_section_block(
						IterIndices{
							Usi:        usi,
							Curriculum: curriculum_idx,
							YearLevel:  year_level_idx,
							Semester:   semester_idx,
							Section:    section_idx,
						},
						IterValues{
							Sched:      sched,
							WeekSched:  week_time_table,
							Curriculum: &curriculum,
							YearLevel:  &year_level,
							Semester:   semester,
						},
					)

					switch irt_section_block {
					case IterBreakCurriculumLoop:
						break curriculum_loop
					}
				}

				usi++
			}
		}
	}
}

func ClearDepartmentSchedule(sched Schedule.UniTimeTables, all_curriculums []Curriculum.Curriculum, department_id uint16, selected_semester int) {
	IterateSectionsWeekSchedule(sched, all_curriculums, selected_semester, nil, nil, func(indecies IterIndices, values IterValues) IterReturnType {

		if values.Curriculum.DepartmentID == department_id {
			values.Sched[indecies.Usi] = Schedule.WeekTimeTable{}
		}

		return IterProceed
	})
}
