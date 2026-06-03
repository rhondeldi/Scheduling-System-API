package RoutesV2

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

type AsyncScheduleResponse struct {
	Records         []Schedule.AsyncScheduleRecord `json:"Records"`
	TotalAsyncHours float64                        `json:"TotalAsyncHours"`
}

// GET /async_schedule?department_id=[N>0]&semester=[0-N>=1][&instructor_id=[N>0]]
func GetAsyncScheduleRecords(ctx *gin.Context) {
	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	department_id, is_valid_department_id_param := RoutesV1.IsValidParameterDepartmentID(ctx)
	if !is_valid_department_id_param {
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, uint16(department_id)); !is_allowed {
		return
	}

	semester, is_valid_semester_param := RoutesV1.IsValidParameterSemesterIndex(ctx)
	if !is_valid_semester_param {
		return
	}

	instructor_id_filter := uint16(0)
	rawInstructorID := ctx.Query("instructor_id")
	if rawInstructorID != "" {
		parsedInstructorID, err := strconv.Atoi(rawInstructorID)
		if err != nil || parsedInstructorID <= 0 {
			ctx.String(http.StatusBadRequest, "invalid instructor_id parameter")
			return
		}
		instructor_id_filter = uint16(parsedInstructorID)
	}

	records, err_read_records := RouteGlobals.ResourcesPersistence.ReaderService.ReadAsyncScheduleRecords(uint16(department_id), semester)
	if err_read_records != nil {
		log.Print("GetAsyncScheduleRecords: read error: ", err_read_records.Error())
		ctx.String(http.StatusInternalServerError, "unable to read asynchronous schedule records right now")
		return
	}

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()
	if err_read_all_curriculums != nil {
		log.Print("GetAsyncScheduleRecords: read curriculums error: ", err_read_all_curriculums.Error())
		ctx.String(http.StatusInternalServerError, "unable to read asynchronous schedule records right now")
		return
	}

	records = RoutesV1.BackfillAsyncRecordCourseSection(records, all_curriculums)

	filtered_records := make([]Schedule.AsyncScheduleRecord, 0, len(records))
	total_async_hours := 0.0

	for _, record := range records {
		if instructor_id_filter > 0 && record.InstructorID != instructor_id_filter {
			continue
		}
		filtered_records = append(filtered_records, record)
		total_async_hours += record.AsyncHours
	}

	ctx.JSON(http.StatusOK, AsyncScheduleResponse{
		Records:         filtered_records,
		TotalAsyncHours: total_async_hours,
	})
}
