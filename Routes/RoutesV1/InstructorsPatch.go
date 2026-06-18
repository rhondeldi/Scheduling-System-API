package RoutesV1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
)

/*
PATCH:

	"/instructor_update"
*/
func PatchInstructor(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	update_instructor_with_time_str := Instructors.InstructorWithTimeString{}

	if err := ctx.BindJSON(&update_instructor_with_time_str); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the instructor updated data")
		return
	}

	selected_instructor, err_read_instructor := RouteGlobals.ResourcesPersistence.ReaderService.ReadInstructor(update_instructor_with_time_str.InstructorID)

	if err_read_instructor != nil {
		ctx.String(http.StatusInternalServerError, "we're unable to find that instructor right now")
		return
	}

	if selected_instructor.DepartmentID != update_instructor_with_time_str.DepartmentID {
		if RouteGlobals.IsGeneratingSchedule.Load() {
			log.Print("PatchInstructor: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
			ctx.String(http.StatusForbidden, "we're unable to move the instructor to other departments right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
			return
		}

		RouteGlobals.ReindexUniSchedMutex.Lock()
		defer RouteGlobals.ReindexUniSchedMutex.Unlock()
	}

	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_departments != nil {
		log.Print("PatchInstructor: ", err_read_departments)
		ctx.String(http.StatusInternalServerError, "we're unable to read the departments needed by the instructor update operation, please try again later.")
		return
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	curriculums, err_read_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_curriculums != nil {
		log.Print("DeleteSubject: [err-read-curriculums] unable to read all curriculums during subject deletion")
		ctx.String(http.StatusForbidden, "we're unable to read all of the curriculums necessary for the subject deletion right now, please try again later.")
		return
	}

	for _, curriculum := range curriculums {
		for _, year_level := range curriculum.YearLevels {
			for _, semester := range year_level.Semesters {
				for _, subject := range semester.Subjects {
					for _, designated_instructor_id := range subject.DesignatedInstructors {
						if !(update_instructor_with_time_str.DepartmentID == selected_instructor.DepartmentID || update_instructor_with_time_str.DepartmentID == 0) {
							if designated_instructor_id == update_instructor_with_time_str.InstructorID {
								if curriculum.DepartmentID == update_instructor_with_time_str.DepartmentID {
									continue
								}

								log.Print("PatchInstructor: [instructor-still-assigned-to-curriculum] unable to move the instructor to other department, that instructor is still designated")
								ctx.String(http.StatusForbidden, fmt.Sprintf(
									"unable to move '%s %s. %s' from %s to %s, still designated by %s in %s, %s, subject %s",
									selected_instructor.FirstName, selected_instructor.MiddleInitial, selected_instructor.LastName,
									dept_id_to_department[selected_instructor.DepartmentID].Code,
									dept_id_to_department[update_instructor_with_time_str.DepartmentID].Code,
									curriculum.CurriculumCode, year_level.Name, semester.Name,
									subject.Code,
								))
								return
							}

						}

					}
				}
			}
		}
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, selected_instructor.DepartmentID); !is_allowed {
		return
	}

	update_instructor := Instructors.Instructor{
		InstructorID:   update_instructor_with_time_str.InstructorID,
		DepartmentID:   update_instructor_with_time_str.DepartmentID,
		FirstName:      update_instructor_with_time_str.FirstName,
		MiddleInitial:  update_instructor_with_time_str.MiddleInitial,
		LastName:       update_instructor_with_time_str.LastName,
		EmploymentType: update_instructor_with_time_str.EmploymentType,
		MaxUnits:       update_instructor_with_time_str.MaxUnits,
		DesignatedSubjectIDs: selected_instructor.DesignatedSubjectIDs,
	}

	if err := update_instructor.Validate(); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	update_instructor.Time.StringParse(update_instructor_with_time_str.Time)

	err := RouteGlobals.ResourcesPersistence.WriterService.UpdateInstructor(update_instructor)

	if err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly updated the instructor")
		return
	}

	ctx.String(http.StatusOK, "instructor updated successfully")
}
