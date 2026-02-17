package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

/*
GET:

	"/university_schedule?semester=[0-1]"
*/
func GetUniversitySchedule(ctx *gin.Context) {

	selected_semester, is_valid_semester_para := IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_para {
		return
	}

	log.Print("GetUniversitySchedule: trying to obtain a university schedule")
	university_schedules, has_obtained := ObtainUniversitySchedule(ctx, nil, selected_semester)

	if !has_obtained {
		log.Print("GetUniversitySchedule: failed to obtain a university schedule")
		return
	}

	log.Print("GetUniversitySchedule: success obtaining university schedule ")

	serialized_schedule := Schedule.SerializeUniversitySchedule(university_schedules)

	log.Print("GetUniversitySchedule: sending university schedule ")

	ctx.Data(http.StatusOK, "application/octet-stream", serialized_schedule)
}
