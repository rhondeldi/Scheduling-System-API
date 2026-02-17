package RoutesV1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type SubjectTablePage struct {
	Subjects      []Curriculum.Subject `json:"Subjects"`
	TotalSubjects int                  `json:"TotalSubjects"`
}

/*
GET:

	"/subjects?page_size=[N>0]&page[0-N>0]&code_match=<string>&name_match=<string>"
*/
func GetSubjects(ctx *gin.Context) {
	page_size, is_valid_page_size_param := IsValidPageSize(ctx)
	if !is_valid_page_size_param {
		return
	}

	page, is_valid_page_param := IsValidPage(ctx)
	if !is_valid_page_param {
		return
	}

	code_match := ctx.Query("code_match")
	name_match := ctx.Query("name_match")

	all_subjects, err_read_all_subjects := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllSubjects()

	if err_read_all_subjects != nil {
		ctx.String(http.StatusInternalServerError, "we're currently unable to get the subjects")
		return
	}

	subjects_page := make([]Curriculum.Subject, 0)

	total_subjects := 0

	for _, subject := range all_subjects {

		if code_match != "" && !Utils.HasSubString(subject.Code, code_match) {
			continue
		}

		if name_match != "" && !Utils.HasSubString(subject.Name, name_match) {
			continue
		}

		total_subjects++

		if (total_subjects - 1) < (page_size * page) {
			continue
		}

		if len(subjects_page) < page_size {
			subjects_page = append(subjects_page, subject)
		}
	}

	subject_table_page := &SubjectTablePage{
		Subjects:      subjects_page,
		TotalSubjects: total_subjects,
	}

	ctx.JSON(http.StatusOK, subject_table_page)
}
