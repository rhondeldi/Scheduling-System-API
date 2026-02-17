package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type CurriculumSectionKey struct {
	YearLevelIndex int
	SemesterIndex  int
	SectionIndex   int
}

/*
PATCH:

	"/curriculum_update"

this route will always fail if the curriculum being edited has no section by default,
so the system should always add a curriculum that has at least one section.
*/
func PatchCurriculum(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	update_curriculum := Curriculum.Curriculum{}

	if err := ctx.BindJSON(&update_curriculum); err != nil {
		log.Print("PatchCurriculum: [json-bind-failed]", err)
		ctx.String(http.StatusBadRequest, "we're unable to read the updated curriculum data")
		return
	}

	// check if the new updated curriculum have at least 1 section

	updated_total_sections := update_curriculum.GetTotalSections()

	if updated_total_sections <= 0 {
		log.Print("PatchCurriculum: updating a curriculum without any sections are not allowed")
		ctx.String(http.StatusBadRequest, "updating a curriculum without any sections are not allowed, a curriculum should have at least 1 section")
		return
	}

	// check if a schedule is still being generated

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("PatchCurriculum: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to update the curriculum right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	// add or remove the section schedule index for the updated curriculum

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we are unable retrieve the curriculums right now")
		return
	}

	for _, curriculum := range all_curriculums {
		if update_curriculum.CurriculumID == curriculum.CurriculumID {
			if is_allowed := Auth.IsDepartmentAllowed(ctx, curriculum.DepartmentID); !is_allowed {
				return
			}
		}
	}

	for selected_semester := range Curriculum.SUPPORTED_SEMESTERS {

		semester_total_sections := update_curriculum.GetTotalSectionsBySemester(selected_semester)

		log.Printf("PatchCurriculum: total sections to be added in %s is %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], semester_total_sections)

		if semester_total_sections <= 0 {
			continue
		}

		// obtain univesity schedules for each semester

		university_schedule, has_obtain := ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)
		updated_university_schedule := make(Schedule.UniTimeTables, 0, len(university_schedule))

		if !has_obtain {
			return
		}

		log.Printf("PatchCurriculum: [read-not-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(university_schedule))

		// determine all the indices of the old untouched to be update curriculum

		curriculum_key_to_weekly_section_sched := make(map[CurriculumSectionKey]Schedule.WeekTimeTable)

		mid_starting_index := -1
		mid_length := 0

		GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedule, all_curriculums, selected_semester, nil, nil,
			func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {
				if values.Curriculum.CurriculumID == update_curriculum.CurriculumID {

					if mid_starting_index == -1 {
						mid_starting_index = indicies.Usi
					}

					curriculum_key_to_weekly_section_sched[CurriculumSectionKey{
						YearLevelIndex: indicies.YearLevel,
						SemesterIndex:  indicies.Semester,
						SectionIndex:   indicies.Section,
					}] = university_schedule[indicies.Usi]

					mid_length++
				}

				return GeneticAlgorithm.IterProceed
			},
		)

		if mid_length == 0 && mid_starting_index == -1 {
			log.Print("PatchCurriculum : [append-to-slice]")

			updated_university_schedule = append(updated_university_schedule, university_schedule...)
			updated_university_schedule = append(updated_university_schedule, make([]Schedule.WeekTimeTable, semester_total_sections)...)
		} else {
			log.Print("PatchCurriculum : [insert-to-slice]")

			// partition university schedules

			uni_sched_left_part, uni_sched_mid_part, uni_sched_right_part, err_midsection_split := Utils.MidSectionSplitInSlice(
				university_schedule, mid_starting_index, mid_length,
			)

			if err_midsection_split != nil {
				log.Print("PatchCurriculum: [mid-section-err]", err_midsection_split)
				ctx.String(http.StatusInternalServerError, "we're unable to update the curriculum right now")
				return
			}

			// rebuild updated curriculum's schedule chunk

			updated_curriculum_schedules := make([]Schedule.WeekTimeTable, 0, len(uni_sched_mid_part))

			for yl_idx, year_level := range update_curriculum.YearLevels {

				if !year_level.IsActive {
					continue
				}

				if selected_semester < 0 || selected_semester >= len(year_level.Semesters) {
					continue // skip invalid semester index
				}

				semester := year_level.Semesters[selected_semester]

				for section_idx := range semester.Sections {

					week_section_sched, has_key := curriculum_key_to_weekly_section_sched[CurriculumSectionKey{
						YearLevelIndex: yl_idx,
						SemesterIndex:  selected_semester,
						SectionIndex:   section_idx,
					}]

					if has_key {
						updated_curriculum_schedules = append(updated_curriculum_schedules, week_section_sched)
					} else {
						updated_curriculum_schedules = append(updated_curriculum_schedules, Schedule.WeekTimeTable{})
					}
				}
			}

			// rebuild the university schedules

			updated_university_schedule = append(updated_university_schedule, uni_sched_left_part...)
			updated_university_schedule = append(updated_university_schedule, updated_curriculum_schedules...)
			updated_university_schedule = append(updated_university_schedule, uni_sched_right_part...)
		}

		// save the new university schedules

		err_save_schedules := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(updated_university_schedule, selected_semester)

		if err_save_schedules != nil {
			ctx.String(http.StatusInternalServerError, "we're unable to save the deletion of the curriculum from the university schedules right now")
			return
		}

		log.Printf("PatchCurriculum: [saved-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(updated_university_schedule))

		err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, updated_university_schedule)

		if err_set_cache != nil {
			log.Print("PatchCurriculum: the new university schedule was saved, but we're unable to cache it")
		}
	}

	//////////////////////////////////////////////////////////////////////

	err := RouteGlobals.ResourcesPersistence.WriterService.UpdateCurriculum(update_curriculum)

	if err != nil {
		log.Print(err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "curriculum updated successfully")
}
