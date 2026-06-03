package RoutesV2

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
)

type InstructorSubjectItem struct {
	SubjectID uint16 `json:"SubjectID"`
	Code      string `json:"Code"`
	Name      string `json:"Name"`
	Units     uint8  `json:"Units"`
}

type instructorSubjectAssignPayload struct {
	InstructorID uint16 `json:"instructor_id"`
	SubjectID    uint16 `json:"subject_id"`

	InstructorIDAlt uint16 `json:"InstructorID"`
	SubjectIDAlt    uint16 `json:"SubjectID"`
}

func resolveAssignIDs(ctx *gin.Context) (uint16, uint16, bool) {
	payload := instructorSubjectAssignPayload{}
	_ = ctx.ShouldBindJSON(&payload)

	instructorID := payload.InstructorID
	subjectID := payload.SubjectID

	if instructorID == 0 {
		instructorID = payload.InstructorIDAlt
	}
	if subjectID == 0 {
		subjectID = payload.SubjectIDAlt
	}

	if instructorID != 0 && subjectID != 0 {
		return instructorID, subjectID, true
	}

	parsedInstructorID, isValidInstructorID := RoutesV1.IsValidInstructorID(ctx)
	if !isValidInstructorID {
		return 0, 0, false
	}

	parsedSubjectID, isValidSubjectID := RoutesV1.IsValidSubjectID(ctx)
	if !isValidSubjectID {
		return 0, 0, false
	}

	return uint16(parsedInstructorID), uint16(parsedSubjectID), true
}

/*
GET:

	"/instructor_subjects?instructor_id=[N>0]"
*/
func GetInstructorSubjects(ctx *gin.Context) {
	instructorID, isValidInstructorID := RoutesV1.IsValidInstructorID(ctx)
	if !isValidInstructorID {
		return
	}

	instructor, errReadInstructor := RouteGlobals.ResourcesPersistence.ReaderService.ReadInstructor(uint16(instructorID))
	if errReadInstructor != nil {
		ctx.String(http.StatusInternalServerError, "unable to read instructor")
		return
	}

	if instructor == nil {
		ctx.String(http.StatusNotFound, "that instructor does not exist")
		return
	}

	if isAllowed := Auth.IsDepartmentAllowed(ctx, instructor.DepartmentID); !isAllowed {
		return
	}

	allSubjects, errReadSubjects := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllSubjects()
	if errReadSubjects != nil {
		ctx.String(http.StatusInternalServerError, "unable to read subjects")
		return
	}

	subjectIDToSubject := make(map[uint16]Curriculum.Subject)
	for _, subject := range allSubjects {
		subjectIDToSubject[subject.ID] = subject
	}

	items := make([]InstructorSubjectItem, 0, len(instructor.DesignatedSubjectIDs))
	for _, designatedSubjectID := range instructor.DesignatedSubjectIDs {
		subject, hasSubject := subjectIDToSubject[designatedSubjectID]
		if !hasSubject {
			continue
		}

		items = append(items, InstructorSubjectItem{
			SubjectID: subject.ID,
			Code:      subject.Code,
			Name:      subject.Name,
			Units:     subject.LecHours + subject.LabHours,
		})
	}

	ctx.JSON(http.StatusOK, items)
}

/*
POST:

	"/instructor_subject_add"
*/
func PostInstructorSubjectAdd(ctx *gin.Context) {
	if isSuccess := Auth.IsAuthSuccess(ctx); !isSuccess {
		return
	}

	instructorID, subjectID, isValid := resolveAssignIDs(ctx)
	if !isValid {
		return
	}

	instructor, errReadInstructor := RouteGlobals.ResourcesPersistence.ReaderService.ReadInstructor(instructorID)
	if errReadInstructor != nil {
		ctx.String(http.StatusInternalServerError, "unable to read instructor")
		return
	}

	if instructor == nil {
		ctx.String(http.StatusNotFound, "that instructor does not exist")
		return
	}

	if isAllowed := Auth.IsDepartmentAllowed(ctx, instructor.DepartmentID); !isAllowed {
		return
	}

	subject, errReadSubject := RouteGlobals.ResourcesPersistence.ReaderService.ReadSubject(subjectID)
	if errReadSubject != nil {
		ctx.String(http.StatusInternalServerError, "unable to read subject")
		return
	}

	if subject == nil {
		ctx.String(http.StatusNotFound, "that subject does not exist")
		return
	}

	for _, designatedSubjectID := range instructor.DesignatedSubjectIDs {
		if designatedSubjectID == subjectID {
			ctx.String(http.StatusOK, "subject already assigned")
			return
		}
	}

	instructor.DesignatedSubjectIDs = append(instructor.DesignatedSubjectIDs, subjectID)
	sort.Slice(instructor.DesignatedSubjectIDs, func(i, j int) bool {
		return instructor.DesignatedSubjectIDs[i] < instructor.DesignatedSubjectIDs[j]
	})

	if errUpdate := RouteGlobals.ResourcesPersistence.WriterService.UpdateInstructor(*instructor); errUpdate != nil {
		ctx.String(http.StatusInternalServerError, "unable to update instructor subject assignment")
		return
	}

	ctx.String(http.StatusOK, "instructor subject assigned")
}

/*
DELETE:

	"/instructor_subject_remove?instructor_id=[N>0]&subject_id=[N>0]"
*/
func DeleteInstructorSubjectRemove(ctx *gin.Context) {
	if isSuccess := Auth.IsAuthSuccess(ctx); !isSuccess {
		return
	}

	instructorIDInt, isValidInstructorID := RoutesV1.IsValidInstructorID(ctx)
	if !isValidInstructorID {
		return
	}

	subjectIDInt, isValidSubjectID := RoutesV1.IsValidSubjectID(ctx)
	if !isValidSubjectID {
		return
	}

	instructorID := uint16(instructorIDInt)
	subjectID := uint16(subjectIDInt)

	instructor, errReadInstructor := RouteGlobals.ResourcesPersistence.ReaderService.ReadInstructor(instructorID)
	if errReadInstructor != nil {
		ctx.String(http.StatusInternalServerError, "unable to read instructor")
		return
	}

	if instructor == nil {
		ctx.String(http.StatusNotFound, "that instructor does not exist")
		return
	}

	if isAllowed := Auth.IsDepartmentAllowed(ctx, instructor.DepartmentID); !isAllowed {
		return
	}

	updatedIDs := make([]uint16, 0, len(instructor.DesignatedSubjectIDs))
	removed := false

	for _, designatedSubjectID := range instructor.DesignatedSubjectIDs {
		if designatedSubjectID == subjectID {
			removed = true
			continue
		}
		updatedIDs = append(updatedIDs, designatedSubjectID)
	}

	if !removed {
		ctx.String(http.StatusNotFound, "subject was not assigned to this instructor")
		return
	}

	instructor.DesignatedSubjectIDs = updatedIDs

	if errUpdate := RouteGlobals.ResourcesPersistence.WriterService.UpdateInstructor(*instructor); errUpdate != nil {
		ctx.String(http.StatusInternalServerError, "unable to update instructor subject assignment")
		return
	}

	ctx.String(http.StatusOK, "instructor subject removed")
}
