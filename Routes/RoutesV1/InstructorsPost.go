package RoutesV1

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
)

/*
POST:

	"/instructor_add"
*/
func PostInstructor(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	add_instructor_with_time_str := Instructors.InstructorWithTimeString{}

	if err := ctx.BindJSON(&add_instructor_with_time_str); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the instructor to be add")
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, add_instructor_with_time_str.DepartmentID); !is_allowed {
		return
	}

	add_instructor := Instructors.Instructor{
		DepartmentID:   add_instructor_with_time_str.DepartmentID,
		FirstName:      add_instructor_with_time_str.FirstName,
		MiddleInitial:  add_instructor_with_time_str.MiddleInitial,
		LastName:       add_instructor_with_time_str.LastName,
		EmploymentType: add_instructor_with_time_str.EmploymentType,
		MaxUnits:       add_instructor_with_time_str.MaxUnits,
	}

	if err := add_instructor.Validate(); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	// reject duplicates: an instructor with the same name (first / middle
	// initial / last, case-insensitive) already in this department.
	existing_instructors, err_read_existing := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllInstructors()
	if err_read_existing != nil {
		log.Print("PostInstructor: [err-read-instructors] ", err_read_existing)
		ctx.String(http.StatusInternalServerError, "we're unable to verify the new instructor against existing instructors right now, please try again later.")
		return
	}

	for _, existing := range existing_instructors {
		if existing.DepartmentID == add_instructor.DepartmentID &&
			strings.EqualFold(strings.TrimSpace(existing.FirstName), add_instructor.FirstName) &&
			strings.EqualFold(strings.TrimSpace(existing.MiddleInitial), add_instructor.MiddleInitial) &&
			strings.EqualFold(strings.TrimSpace(existing.LastName), add_instructor.LastName) {
			ctx.String(http.StatusConflict, fmt.Sprintf(
				"an instructor named '%s %s. %s' already exists in this department",
				add_instructor.FirstName, add_instructor.MiddleInitial, add_instructor.LastName,
			))
			return
		}
	}

	add_instructor.Time.StringParse(add_instructor_with_time_str.Time)

	err := RouteGlobals.ResourcesPersistence.WriterService.CreateInstructor(add_instructor)

	if err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly add the instructor")
		return
	}

	ctx.String(http.StatusOK, "instructor added successfully")
}
