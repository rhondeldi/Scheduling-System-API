package RoutesV2

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

/*
GET:

	"/v2/clear_class_schedule?department_id=[N>0]&semester=[0-N>=1]&curriculum_id=[N>0]&year_level_idx=[0-N>=1]&section_idx=[0-N>=1]"
*/
func DeleteClearClassSchedule(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("v2.DeleteClearClassSchedule: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to clear the class schedule right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	// parse curriculum id

	curriculum_id, is_valid_curriculum_id_param := RoutesV1.IsValidCurriculumID(ctx)

	if !is_valid_curriculum_id_param {
		return
	}

	// parse year level index

	param_year_level_idx, is_valid_year_level_idx_param := RoutesV1.IsValidIndex(ctx, "year_level_idx")

	if !is_valid_year_level_idx_param {
		return
	}

	// parse section index

	param_section_idx, is_valid_section_idx_param := RoutesV1.IsValidIndex(ctx, "section_idx")

	if !is_valid_section_idx_param {
		return
	}

	// parse semester parameter

	selected_semester, is_valid_semester_param := RoutesV1.IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_param {
		return
	}

	// parse department_id parameter

	department_id, is_valid_department_id_param := RoutesV1.IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id_param {
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, uint16(department_id)); !is_allowed {
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	// load university schedules

	university_schedules, has_obtained := RoutesV1.ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)

	if !has_obtained {
		return
	}

	// cache the found university schedule for the semester

	err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, university_schedules)

	if err_set_cache != nil {
		log.Println(err_set_cache.Error())
	}

	// get all curriculums

	all_curriculums, err_read_all_curriculum := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		ctx.String(http.StatusInternalServerError, "unable to read curriculums for that department")
		return
	}

	// parse schedule_idx parameter

	schedule_idx := 0

curriculum_loop:
	for _, curriculum := range all_curriculums {
		for year_level_idx, year_level := range curriculum.YearLevels {
			if !year_level.IsActive {
				continue
			}

			for semester_idx, semester := range year_level.Semesters {
				if semester_idx != selected_semester {
					continue
				}

				for section_idx := 0; section_idx < semester.Sections; section_idx++ {

					if curriculum_id == int(curriculum.CurriculumID) && year_level_idx == param_year_level_idx && section_idx == param_section_idx {
						break curriculum_loop
					}

					schedule_idx++
				}
			}
		}
	}

	// clear class schedule

	university_schedules[schedule_idx] = Schedule.WeekTimeTable{}

	// save schedule

	err_save_schedules := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(university_schedules, selected_semester)

	if err_save_schedules != nil {
		log.Print("DeleteClearClassSchedule: (save error) ", err_save_schedules.Error())
		ctx.String(http.StatusOK, "we're unable to clear that schedule's week time table")
		return
	}

	ctx.String(http.StatusOK, "weekly time table schedule was successfully cleared")
}
