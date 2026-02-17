package RoutesV1

import (
	"log"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

/*
POST:

	"/curriculum_add"
*/
func PostCurriculum(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	add_curriculum := Curriculum.Curriculum{}

	if err := ctx.BindJSON(&add_curriculum); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the curriculum to be added")
		return
	}

	// check if the new updated curriculum have at least 1 section

	add_total_sections := add_curriculum.GetTotalSections()

	if add_total_sections <= 0 {
		log.Print("PostCurriculum: add a curriculum without any sections are not allowed")
		ctx.String(http.StatusBadRequest, "adding a curriculum without any sections are not allowed, a curriculum should have at least 1 section")
		return
	}

	// check if a schedule is still being generated

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("PostCurriculum: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to add a curriculum right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	// get current curriculums

	curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we are unable retrieve the curriculums right now")
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, add_curriculum.DepartmentID); !is_allowed {
		return
	}

	// obtain current university schedules for each semester

	schedules_for_each_semester := make([]Schedule.UniTimeTables, 0, Curriculum.SUPPORTED_SEMESTERS)

	for selected_semester := range Curriculum.SUPPORTED_SEMESTERS {
		university_schedule, has_obtain := ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)

		if !has_obtain {
			return
		}

		log.Printf("PostCurriculum: [read-not-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(university_schedule))

		schedules_for_each_semester = append(schedules_for_each_semester, university_schedule)
	}

	// save new curriculum

	new_curriculum_id, err := RouteGlobals.ResourcesPersistence.WriterService.CreateCurriculum(add_curriculum)

	if err != nil {
		log.Print("PostCurriculum: [save-error] ", err)
		ctx.String(http.StatusBadRequest, "we are unable to properly add the curriculum")
		return
	}

	if new_curriculum_id == 0 {
		log.Print("PostCurriculum: [save-id-error] detected a new curriculum with an id of 0")
		ctx.String(http.StatusBadRequest, "we are unable to properly add the curriculum, the system generated an unknown id")
		return
	}

	// add the new curriculum to the in memory curriculums

	add_curriculum.CurriculumID = new_curriculum_id
	curriculums = append(curriculums, add_curriculum)

	sort.Slice(curriculums, func(i, j int) bool {
		return curriculums[i].CurriculumID < curriculums[j].CurriculumID
	})

	// rebuild university schedule index

	for selected_semester := range Curriculum.SUPPORTED_SEMESTERS {

		new_sections := add_curriculum.GetTotalSectionsBySemester(selected_semester)

		if new_sections <= 0 {
			continue
		}

		university_schedule := schedules_for_each_semester[selected_semester]

		// determine insert university schedule index for the new curriculum

		insert_idx := -1
		insert_length := 0

		GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedule, curriculums, selected_semester, nil, nil,
			func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {

				is_equal_code := Utils.IsEqualStrCaseInsensitiveIgnoreWhiteSpace(values.Curriculum.CurriculumCode, add_curriculum.CurriculumCode)
				is_equal_name := Utils.IsEqualStrCaseInsensitiveIgnoreWhiteSpace(values.Curriculum.CurriculumName, add_curriculum.CurriculumName)

				if is_equal_code && is_equal_name {
					if insert_idx == -1 {
						insert_idx = indicies.Usi
					}

					insert_length++

				}

				if indicies.Usi >= (len(university_schedule) - 1) {
					return GeneticAlgorithm.IterBreakCurriculumLoop
				}

				return GeneticAlgorithm.IterProceed
			},
		)

		if insert_idx > len(university_schedule) {
			log.Printf(
				"PostCurriculum: [fatal-error] insert_idx %d exceeds university_schedule length %d, the iteration function might be broken",
				insert_idx, len(university_schedule),
			)

			ctx.String(
				http.StatusInternalServerError,
				"we're unable to rebuild the university schedule right now after adding the curriculum due to an internal iteration error",
			)

			delete_curriculum_on_failure(ctx, new_curriculum_id)
			return
		}

		// re-build index of the new university schedule with the new curriculum

		new_university_schedule := make(Schedule.UniTimeTables, 0, len(university_schedule))

		if insert_idx == len(university_schedule) {
			new_university_schedule = append(new_university_schedule, university_schedule...)
			new_university_schedule = append(new_university_schedule, make(Schedule.UniTimeTables, new_sections)...)

			log.Printf(
				"PostCurriculum: [rebuilt-index-last-append] new %d section(s) are added to the university schedule %s",
				new_sections, Curriculum.SEMESTER_INDEX_NAME[selected_semester],
			)
		} else if insert_idx < 0 {
			new_university_schedule = append(new_university_schedule, university_schedule...)
			new_university_schedule = append(new_university_schedule, make(Schedule.UniTimeTables, new_sections)...)

			log.Printf(
				"PostCurriculum: [rebuilt-index-last-append???????????????????????] new %d section(s) are added to the university schedule %s",
				new_sections, Curriculum.SEMESTER_INDEX_NAME[selected_semester],
			)
		} else {

			if new_sections != insert_length {
				log.Printf(
					"PostCurriculum: [rebuilt-index-length-error] new sections %d does not match the insert length %d",
					new_sections, insert_length,
				)

				ctx.String(
					http.StatusInternalServerError,
					"we're unable to rebuild the university schedule right now after adding the curriculum due to an internal length error",
				)

				delete_curriculum_on_failure(ctx, new_curriculum_id)
				return
			}

			new_university_schedule = append(new_university_schedule, university_schedule[:insert_idx]...)
			new_university_schedule = append(new_university_schedule, make(Schedule.UniTimeTables, insert_length)...)
			new_university_schedule = append(new_university_schedule, university_schedule[insert_idx:]...)

			log.Printf(
				"PostCurriculum: [rebuilt-index-insert] new %d section(s) are added to the university schedule %s, starting index %d",
				new_sections, Curriculum.SEMESTER_INDEX_NAME[selected_semester], insert_idx,
			)
		}

		// save the new university schedules

		err_save_schedules := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(new_university_schedule, selected_semester)
		log.Printf("PostCurriculum: [save-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(new_university_schedule))

		if err_save_schedules != nil {
			log.Print("PostCurriculum: [uni-sched-save-error] caused by ", err_save_schedules.Error())
			ctx.String(http.StatusInternalServerError, "we're unable to save the curriculum, re-indexed university schedules save failed")
			delete_curriculum_on_failure(ctx, new_curriculum_id)
			return
		}

		err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, new_university_schedule)

		if err_set_cache != nil {
			log.Print("PostCurriculum: the new university schedule was saved, but we're unable to cache it")
		}
	}

	ctx.String(http.StatusOK, "curriculum added successfully")
}

func delete_curriculum_on_failure(
	ctx *gin.Context,
	curriculum_id uint16,
) {
	err := RouteGlobals.ResourcesPersistence.WriterService.DeleteCurriculum(curriculum_id)

	if err != nil {
		log.Print("PostCurriculum: [fatal-error] unable to delete the curriculum after failure")
		ctx.String(http.StatusInternalServerError,
			" : fatal error, unable to delete the curriculum after failure, this could break whole current university schedules",
		)
		return
	}

	log.Print("PostCurriculum: [recover-successful] unable to add the curriculum, but the curriculum was deleted successfully")
}
