package RoutesV1

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

/*
POST:

	"/university_schedule"
*/
func PostUniversitySchedule(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	////////////////////////////////////////////////////////////////////////////////////////
	//                                   FOR TESTING
	////////////////////////////////////////////////////////////////////////////////////////

	selected_semester, is_valid_semester_para := IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_para {
		return
	}

	university_schedules, err_load_schedules := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(selected_semester)

	if err_load_schedules != nil {
		ctx.String(http.StatusInternalServerError, "error reading the schedule")
		return
	}

	if university_schedules.IsEmpty() {
		ctx.String(http.StatusNotFound, "schedule was empty, please generate one first")
		return
	}

	rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		log.Print("PostUniversitySchedule: [read-rooms-error] caused by ", err_read_all_rooms)
		ctx.String(http.StatusInternalServerError, "tried to add university schedule but, we can not retrieve the required rooms information right now")
		return
	}

	curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_curriculums != nil {
		log.Print("PostUniversitySchedule: [read-curriculums-error] caused by ", err_curriculums)
		ctx.String(http.StatusInternalServerError, "tried to add university schedule but, we can not retrieve the required curriculums information right now")
		return
	}

	for _, err_vertical_validation := range university_schedules.VerticalValidation(rooms) {
		if err_vertical_validation != nil {
			ctx.String(http.StatusConflict, "we detected an invalid schedule")
			return
		}
	}

	errs_horizontal_validation := GeneticAlgorithm.HorizontalValidation(
		university_schedules, curriculums, nil, selected_semester,
	)

	for _, err_horizontal_validation := range errs_horizontal_validation {
		if err_horizontal_validation != nil {
			ctx.String(http.StatusConflict, "we detected an invalid schedule")
			return
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////

	serialized_data, err_read_all := io.ReadAll(ctx.Request.Body)

	if err_read_all != nil {
		ctx.String(http.StatusBadRequest, "we failed to read that post request body")
		return
	}

	received_university_schedules := Schedule.DeserializeUniversitySchedule(serialized_data)

	if len(university_schedules) != len(received_university_schedules) {
		ctx.String(http.StatusConflict, "schedule length mismatch")
		return
	}

	for section_idx := 0; section_idx < len(university_schedules); section_idx++ {
		for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
			for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
				if university_schedules[section_idx][day][time_slot].GetSubjectID() != received_university_schedules[section_idx][day][time_slot].GetSubjectID() {
					ctx.String(http.StatusConflict, "schedule subject id mismatch")
					return
				}

				if university_schedules[section_idx][day][time_slot].GetInstructorID() != received_university_schedules[section_idx][day][time_slot].GetInstructorID() {
					ctx.String(http.StatusConflict, "schedule instructor id mismatch")
					return
				}

				if university_schedules[section_idx][day][time_slot].GetRoomID() != received_university_schedules[section_idx][day][time_slot].GetRoomID() {
					ctx.String(http.StatusConflict, "schedule room id mismatch")
					return
				}
			}
		}
	}

	ctx.String(http.StatusOK, "schedule successfully matched")
}
