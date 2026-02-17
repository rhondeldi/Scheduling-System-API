package RoutesV1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type CurriculumPageItem struct {
	CurriculumID   uint16 `json:"CurriculumID"` // non negative & non-zero unique number even if for example we have Computer Science (OLD) and Computer Science (New)
	DepartmentID   uint16 `json:"DepartmentID"`
	CurriculumName string `json:"CurriculumName"` // e.g. Computer Science, Information Technology
	CurriculumCode string `json:"CurriculumCode"` // e.g. BSCS, BSIT
}

type CurriculumTablePage struct {
	Curriculums      []CurriculumPageItem `json:"Curriculums"`
	TotalCurriculums int                  `json:"TotalCurriculums"`
}

/*
GET:

	"/curriculum_list?page_size=[N>0]&page=[0-N>0]&department_id=[N>=0]&code_match=<string>&name_match=<string>"
*/
func GetDepartmentCurriculumList(ctx *gin.Context) {
	page_size, is_valid_page_size_param := IsValidPageSize(ctx)
	if !is_valid_page_size_param {
		return
	}

	page, is_valid_page_param := IsValidPage(ctx)
	if !is_valid_page_param {
		return
	}

	department_id, is_valid_department_id_param := IsValidParameterDepartmentID(ctx)
	if !is_valid_department_id_param {
		return
	}

	code_match := ctx.Query("code_match")
	name_match := ctx.Query("name_match")

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()
	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we're currently unable to get the curriculums")
		return
	}

	curriculums_page := make([]CurriculumPageItem, 0)
	total_curriculums := 0

	for _, curriculum := range all_curriculums {
		if int(curriculum.DepartmentID) != department_id {
			continue
		}

		if code_match != "" && !Utils.HasSubString(curriculum.CurriculumCode, code_match) {
			continue
		}

		if name_match != "" && !Utils.HasSubString(curriculum.CurriculumName, name_match) {
			continue
		}

		total_curriculums++

		if (total_curriculums - 1) < (page_size * page) {
			continue
		}

		if len(curriculums_page) < page_size {
			curriculums_page = append(curriculums_page, CurriculumPageItem{
				CurriculumID:   curriculum.CurriculumID,
				DepartmentID:   curriculum.DepartmentID,
				CurriculumName: curriculum.CurriculumName,
				CurriculumCode: curriculum.CurriculumCode,
			})
		}
	}

	curriculum_table_page := &CurriculumTablePage{
		Curriculums:      curriculums_page,
		TotalCurriculums: total_curriculums,
	}

	ctx.JSON(http.StatusOK, curriculum_table_page)
}

/*
GET:

	"/curriculum_load?curriculum_id=[N>0]"
*/
func GetCurriculum(ctx *gin.Context) {
	curriculum_id, is_valid_curriculum_id_param := IsValidCurriculumID(ctx)

	if !is_valid_curriculum_id_param {
		return
	}

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we're currently unable to get the curriculums")
		return
	}

	for _, curriculum := range all_curriculums {
		if curriculum.CurriculumID == uint16(curriculum_id) {
			ctx.JSON(http.StatusOK, curriculum)
			return
		}
	}

	ctx.String(http.StatusNotFound, "that curriculum id does not exist yet")
}
