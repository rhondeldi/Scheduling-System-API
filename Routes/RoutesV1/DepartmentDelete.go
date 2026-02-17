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
)

/*
DELETE:

	"/department_remove?department_id=[N>0]"
*/
func DeleteDepartment(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	department_id, is_valid_department_id_param := IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id_param {
		return
	}

	if department_id == 0 {
		ctx.String(http.StatusForbidden, "deleting the general department is now allowed")
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, uint16(department_id)); !is_allowed {
		return
	}

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("DeleteDepartment: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to delete the department right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		log.Print("DeleteDepartment: [error curriculum reads] error reading all curriculums")
		ctx.String(http.StatusInternalServerError, "we are unable retrieve the curriculums right now")
		return
	}

	for selected_semester := range Curriculum.SUPPORTED_SEMESTERS {
		log.Printf("DeleteDepartment: [re-index-semester] rebuilding university schedule index for the %s", Curriculum.SEMESTER_INDEX_NAME[selected_semester])

		// obtain univesity schedules for each semester

		new_university_schedule := make(Schedule.UniTimeTables, 0)
		university_schedule, has_obtain := ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)

		if !has_obtain {
			return
		}

		log.Printf("DeleteDepartment: [read-not-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(university_schedule))

		GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedule, all_curriculums, selected_semester, nil, nil,
			func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {
				if values.Curriculum.DepartmentID != uint16(department_id) {
					new_university_schedule = append(new_university_schedule, *values.WeekSched)
				}

				return GeneticAlgorithm.IterProceed
			},
		)

		// save the new university schedules

		err_save_schedules := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(new_university_schedule, selected_semester)

		if err_save_schedules != nil {
			log.Print("DeleteDepartment: error save schedule")
			ctx.String(http.StatusInternalServerError, "we're unable to save the deletion of the curriculum from the university schedules right now")
			return
		}

		log.Printf("DeleteDepartment: [saved-modified] university schedule length for the %s : %d", Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(new_university_schedule))

		err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, new_university_schedule)

		if err_set_cache != nil {
			log.Print("DeleteDepartment: the new university schedule was saved, but we're unable to cache it")
		}
	}

	err := RouteGlobals.ResourcesPersistence.WriterService.DeleteDepartment(uint16(department_id))

	if err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly remove the department")
		return
	}

	// delete all curriculums for that department

	for _, curriculum := range all_curriculums {
		if curriculum.DepartmentID == uint16(department_id) {
			err_delete_curriculum := RouteGlobals.ResourcesPersistence.WriterService.DeleteCurriculum(curriculum.CurriculumID)

			if err_delete_curriculum != nil {
				log.Printf("DeleteDepartment: error deleting curriculum %s", curriculum.CurriculumName)

				ctx.String(
					http.StatusInternalServerError,
					"fatal error, we're unable to delete the associated curriculum of that department, the developer an email to fix this error mrdcvlsc@gmail.com",
				)

				return
			}
		}
	}

	ctx.String(http.StatusOK, "department deleted successfully")
}
