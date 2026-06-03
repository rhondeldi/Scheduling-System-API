package RoutesV1

import (
	"fmt"
	"sort"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

func BuildDepartmentAsyncScheduleRecords(
	university_schedules Schedule.UniTimeTables,
	all_curriculums []Curriculum.Curriculum,
	all_subjects []Curriculum.Subject,
	department_id uint16,
	semester int,
) []Schedule.AsyncScheduleRecord {
	subject_id_to_subject := make(map[uint16]Curriculum.Subject)
	for _, subject := range all_subjects {
		normalized := subject
		normalized.NormalizeAsyncConfig()
		subject_id_to_subject[subject.ID] = normalized
	}

	records := make([]Schedule.AsyncScheduleRecord, 0)
	currentYear := time.Now().Year()

	GeneticAlgorithm.IterateSectionsWeekSchedule(
		university_schedules, all_curriculums, semester, nil, nil,
		func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {
			if values.Curriculum == nil || values.Semester == nil || values.WeekSched == nil {
				return GeneticAlgorithm.IterProceed
			}

			if values.Curriculum.DepartmentID != department_id {
				return GeneticAlgorithm.IterProceed
			}

			subject_id_to_instructor_id := make(map[uint16]uint16)
			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for timeSlot := 0; timeSlot < Const.N_DAILY_TIME_SLOTS; timeSlot++ {
					slot := values.WeekSched[day][timeSlot]
					subjectID := slot.GetSubjectID()
					instructorID := slot.GetInstructorID()
					if subjectID == 0 || instructorID == 0 {
						continue
					}
					if subject_id_to_instructor_id[subjectID] == 0 {
						subject_id_to_instructor_id[subjectID] = instructorID
					}
				}
			}

			for _, curriculumSubject := range values.Semester.Subjects {
				subject, hasSubject := subject_id_to_subject[curriculumSubject.ID]
				if !hasSubject {
					continue
				}

				asyncHours := subject.EffectiveAsynchronousHours()
				if asyncHours <= 0 {
					continue
				}

				sectionLetter := ""
				if indicies.Section >= 0 && indicies.Section < len(Curriculum.SECTION) {
					sectionLetter = Curriculum.SECTION[indicies.Section]
				}
				courseSection := fmt.Sprintf(
					"%s-%d%s",
					values.Curriculum.CurriculumCode,
					indicies.YearLevel+1,
					sectionLetter,
				)

				displayLabel := fmt.Sprintf("%s (Async)", subject.Code)
				sectionID := fmt.Sprintf(
					"%d-%d-%d-%d",
					values.Curriculum.CurriculumID,
					indicies.YearLevel,
					indicies.Section,
					indicies.Usi,
				)

				record := Schedule.AsyncScheduleRecord{
					SectionID:     sectionID,
					SectionUSI:    indicies.Usi,
					CurriculumID:  values.Curriculum.CurriculumID,
					DepartmentID:  values.Curriculum.DepartmentID,
					YearLevelIdx:  indicies.YearLevel,
					SectionIdx:    indicies.Section,
					Semester:      semester,
					Year:          currentYear,
					SubjectID:     subject.ID,
					InstructorID:  subject_id_to_instructor_id[subject.ID],
					AsyncHours:    asyncHours,
					CourseSection: courseSection,
					DisplayLabel:  displayLabel,
				}

				records = append(records, record)
			}

			return GeneticAlgorithm.IterProceed
		},
	)

	sort.Slice(records, func(i, j int) bool {
		if records[i].InstructorID != records[j].InstructorID {
			return records[i].InstructorID < records[j].InstructorID
		}
		if records[i].SubjectID != records[j].SubjectID {
			return records[i].SubjectID < records[j].SubjectID
		}
		return records[i].SectionUSI < records[j].SectionUSI
	})

	return records
}

// BackfillAsyncRecordCourseSection populates the CourseSection field on records
// that were persisted before the field existed. Records that already have a
// CourseSection are returned unchanged. Used by GET endpoints so old data
// renders with the standard "BSCS-1D" style without requiring regeneration.
func BackfillAsyncRecordCourseSection(records []Schedule.AsyncScheduleRecord, all_curriculums []Curriculum.Curriculum) []Schedule.AsyncScheduleRecord {
	if len(records) == 0 {
		return records
	}

	curriculum_id_to_code := make(map[uint16]string, len(all_curriculums))
	for _, curriculum := range all_curriculums {
		curriculum_id_to_code[curriculum.CurriculumID] = curriculum.CurriculumCode
	}

	for i := range records {
		if records[i].CourseSection != "" {
			continue
		}

		sectionLetter := ""
		if records[i].SectionIdx >= 0 && records[i].SectionIdx < len(Curriculum.SECTION) {
			sectionLetter = Curriculum.SECTION[records[i].SectionIdx]
		}

		records[i].CourseSection = fmt.Sprintf(
			"%s-%d%s",
			curriculum_id_to_code[records[i].CurriculumID],
			records[i].YearLevelIdx+1,
			sectionLetter,
		)
	}

	return records
}

func RegenerateDepartmentAsyncScheduleRecords(
	university_schedules Schedule.UniTimeTables,
	all_curriculums []Curriculum.Curriculum,
	department_id uint16,
	semester int,
) error {
	all_subjects, err_read_all_subjects := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllSubjects()
	if err_read_all_subjects != nil {
		return err_read_all_subjects
	}

	records := BuildDepartmentAsyncScheduleRecords(
		university_schedules,
		all_curriculums,
		all_subjects,
		department_id,
		semester,
	)

	return RouteGlobals.ResourcesPersistence.WriterService.ReplaceAsyncScheduleRecords(department_id, semester, records)
}
