package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

/*
DELETE:

	"/curriculum_remove?curriculum_id=[N>0]"
*/
func DeleteCurriculum(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	curriculum_id, is_valid_curriculum_id_param := IsValidCurriculumID(ctx)

	if !is_valid_curriculum_id_param {
		return
	}

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("DeleteCurriculum: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to delete the curriculum right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		log.Print("DeleteCurriculum: [error curriculum reads] error reading all curriculums")
		ctx.String(http.StatusInternalServerError, "we are unable retrieve the curriculums right now")
		return
	}

	// auth department

	for _, curriculum := range all_curriculums {
		if curriculum_id == int(curriculum.CurriculumID) {
			if is_allowed := Auth.IsDepartmentAllowed(ctx, curriculum.DepartmentID); !is_allowed {
				log.Print("DeleteCurriculum: [department not allowed] error reading all curriculums")
				return
			}
		}
	}

	for selected_semester := range Curriculum.SUPPORTED_SEMESTERS {
		log.Printf("DeleteCurriculum: [re-index-semester] rebuilding university schedule index for the %s", Curriculum.SEMESTER_INDEX_NAME[selected_semester])

		// obtain univesity schedules for each semester

		university_schedule, has_obtain := ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)

		if !has_obtain {
			return
		}

		log.Printf("DeleteCurriculum: [read-not-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(university_schedule))

		// determine which schedule indices should be removed from the current university schedules

		remove_starting_index := -1
		remove_chunk_length := 0

		GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedule, all_curriculums, selected_semester, nil, nil,
			func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {
				if values.Curriculum != nil && values.Curriculum.CurriculumID == uint16(curriculum_id) {

					if remove_starting_index == -1 {
						remove_starting_index = indicies.Usi
					}

					remove_chunk_length++
				}

				return GeneticAlgorithm.IterProceed
			},
		)

		if remove_chunk_length == 0 {
			continue
		}

		// Validate indices are within bounds before attempting removal
		if remove_starting_index < 0 || remove_starting_index >= len(university_schedule) || (remove_starting_index+remove_chunk_length) > len(university_schedule) {
			log.Printf("DeleteCurriculum: [skip-semester] curriculum sections are out of bounds for semester %s: start=%d, size=%d, schedule_len=%d. Likely the schedule doesn't have entries for this curriculum yet.", Curriculum.SEMESTER_INDEX_NAME[selected_semester], remove_starting_index, remove_chunk_length, len(university_schedule))
			continue
		}

		// remove the to be deleted schedule indices from the university schedules

		log.Printf("DeleteCurriculum: removing university schedule index %d to %d (of size %d)", remove_starting_index, remove_starting_index+remove_chunk_length, remove_chunk_length)

		new_university_schedule, err_remove_chunk_in_slice := Utils.RemoveChunkInSlice(university_schedule, remove_starting_index, remove_chunk_length)

		if err_remove_chunk_in_slice != nil {
			log.Print("DeleteCurriculum: error remove chunk in slice, caused by : ", err_remove_chunk_in_slice.Error())
			ctx.String(http.StatusInternalServerError, "we're unable to remove the curriculum to the university schedules right now")
			return
		}

		// save the new university schedules

		err_save_schedules := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(new_university_schedule, selected_semester)

		if err_save_schedules != nil {
			log.Print("DeleteCurriculum: error save schedule")
			ctx.String(http.StatusInternalServerError, "we're unable to save the deletion of the curriculum from the university schedules right now")
			return
		}

		log.Printf("DeleteCurriculum: [saved-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(new_university_schedule))

		err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, new_university_schedule)

		if err_set_cache != nil {
			log.Print("DeleteCurriculum: the new university schedule was saved, but we're unable to cache it")
		}
	}

	err := RouteGlobals.ResourcesPersistence.WriterService.DeleteCurriculum(uint16(curriculum_id))

	if err != nil {
		log.Print("DeleteCurriculum: error delete curriculum")
		ctx.String(http.StatusBadRequest, "we are unable to properly remove the curriculum from the persistence")
		return
	}

	ctx.String(http.StatusOK, "curriculum deleted successfully")
}
