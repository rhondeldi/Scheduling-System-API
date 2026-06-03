package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

/*
GET:

	"/v1/clear_department_schedules?department_id=[N>0]&semester=[0-1]"
*/
func DeleteClearDepartmentSchedule(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	// parse semester parameter

	semester, is_valid_semester_param := IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_param {
		return
	}

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("DeleteClearDepartmentSchedule: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to clear the department schedules right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	// parse department_id parameter

	department_id, is_valid_department_id_param := IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id_param {
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, uint16(department_id)); !is_allowed {
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	// load university schedules

	university_schedules, has_obtained := ObtainUniversityScheduleNoHorizontalValidation(ctx, semester)

	if !has_obtained {
		return
	}

	// cache found for university schedule for the current semester

	err_set_cache := RouteGlobals.SetCachedUniversitySchedule(semester, university_schedules)

	if err_set_cache != nil {
		log.Println(err_set_cache.Error())
	}

	// get all curriculums

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we are unable retrieve the curriculums right now")
		return
	}

	// clear class schedule

	schedule_idx := 0

	for _, curriculum := range all_curriculums {
		for _, year_level := range curriculum.YearLevels {

			if !year_level.IsActive {
				continue
			}

			for semester_idx, semester_element := range year_level.Semesters {
				if semester_idx != semester {
					continue
				}

				for section_idx := 0; section_idx < semester_element.Sections; section_idx++ {

					if curriculum.DepartmentID == uint16(department_id) {
						university_schedules[schedule_idx] = Schedule.WeekTimeTable{}
					}

					schedule_idx++
				}
			}
		}
	}

	// save schedule

	err_save_schedules := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(university_schedules, semester)

	if err_save_schedules != nil {
		log.Print("DeleteClearClassSchedule: (save error) ", err_save_schedules.Error())
		ctx.String(http.StatusOK, "we're unable to clear the department schedules")
		return
	}

	if err_regen_async := RegenerateDepartmentAsyncScheduleRecords(
		university_schedules,
		all_curriculums,
		uint16(department_id),
		semester,
	); err_regen_async != nil {
		log.Print("DeleteClearDepartmentSchedule: [async-records-failed] ", err_regen_async.Error())
		ctx.String(http.StatusInternalServerError, "department schedules were cleared but async schedule records could not be refreshed")
		return
	}

	RouteGlobals.SetDeptSchedGenResult(
		RouteGlobals.DeptSchedGenKey{
			DepartmentID: uint16(department_id),
			Semester:     semester,
		},
		RouteGlobals.SchedGenResult{
			Status:  RouteGlobals.SchedGenStatusNotStarted,
			Message: "department schedules was deleted in the last action",
		},
	)

	ctx.String(http.StatusOK, "all department schedule was successfully cleared")
}

/*
GET:

	"/v1/delete_all_generated_university_schedules_for_all_semester_a_complete_reset"
*/
func DeleteAllUniversitySchedules(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("DeleteAllUniversitySchedules: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "please wait for other department to finish generating schedules")
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	RouteGlobals.ClearCachedUniversitySchedule()

	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()
	if err_read_departments != nil {
		log.Print("DeleteAllUniversitySchedules: (read departments error) ", err_read_departments.Error())
		ctx.String(http.StatusInternalServerError, "unable to clear async schedule records right now")
		return
	}

	for semester := range Curriculum.SUPPORTED_SEMESTERS {

		// delete schedule

		err_delete_schedules := RouteGlobals.SchedulePersistence.SaveService.DeleteSchedules(semester)

		if err_delete_schedules != nil {
			log.Print("DeleteClearClassSchedule: (save error) ", err_delete_schedules.Error())
			ctx.String(http.StatusOK, "we're unable to clear the department schedules")
			return
		}

		for _, department := range departments {
			if err_delete_async := RouteGlobals.ResourcesPersistence.WriterService.DeleteAsyncScheduleRecords(department.DepartmentID, semester); err_delete_async != nil {
				log.Print("DeleteAllUniversitySchedules: (delete async records error) ", err_delete_async.Error())
				ctx.String(http.StatusInternalServerError, "unable to clear async schedule records")
				return
			}
		}
	}

	ctx.String(http.StatusOK, "university schedules was successfully deleted")
}
