package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
)

/*
GET:

	"/gen_status"
*/
func GetGenStatus(ctx *gin.Context) {
	status := make(map[string]bool)
	status["status"] = RouteGlobals.IsGeneratingSchedule.Load()

	log.Printf("GetGenStatus: call result %t %t", status["status"], RouteGlobals.IsGeneratingSchedule.Load())

	ctx.JSON(http.StatusOK, status)
}

/*
GET:

	"/dept_gen_result?department_id=[N>0]&semester=[0-1]"
*/
func GetDeptartmentGenerationResult(ctx *gin.Context) {
	department_id, is_valid_department_id := IsValidParameterDepartmentID(ctx)
	if !is_valid_department_id {
		log.Print("GetDeptartmentGenerationResult: invalid department id")
		ctx.String(http.StatusBadRequest, "invalid department id")
		return
	}

	semester_index, is_valid_semester := IsValidParameterSemesterIndex(ctx)
	if !is_valid_semester {
		log.Print("GetDeptartmentGenerationResult: invalid semester index")
		ctx.String(http.StatusBadRequest, "invalid semester index")
		return
	}

	last_result := RouteGlobals.GetDepartSchedGenResult(
		RouteGlobals.DeptSchedGenKey{
			DepartmentID: uint16(department_id),
			Semester:     semester_index,
		},
	)

	ctx.JSON(http.StatusOK, last_result)
}
