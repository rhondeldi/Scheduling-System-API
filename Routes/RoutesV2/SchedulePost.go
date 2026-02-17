package RoutesV2

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type WeekTimeTableSubjects []RoutesV1.SubjectAssignmentInfo

/*
POST:

	"/add_schedule_preference"
*/
func PostWeekTimeTableSurvery(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	var configured_week_time_table []RoutesV1.SubjectAssignmentInfo

	if err := ctx.BindJSON(&configured_week_time_table); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the prefered configured week time table")
		return
	}

	Utils.PrettyPrint(configured_week_time_table)

	///////

	var configured_week_time_table_records [][]RoutesV1.SubjectAssignmentInfo

	file_data, err := os.ReadFile("week-time-table-scedule-preferences.json")

	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("error reading file: %v\n", err)
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		configured_week_time_table_records = make([][]RoutesV1.SubjectAssignmentInfo, 0)
	} else {
		// File exists — try to parse the JSON
		if err := json.Unmarshal(file_data, &configured_week_time_table_records); err != nil {
			log.Printf("error parsing JSON: %v\n", err)
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	configured_week_time_table_records = append(configured_week_time_table_records, configured_week_time_table)

	updated, err := json.MarshalIndent(configured_week_time_table_records, "", " ")

	if err != nil {
		log.Printf("error marshaling: %v\n", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	if err := os.WriteFile("week-time-table-scedule-preferences.json", updated, 0644); err != nil {
		log.Printf("error writing file: %v\n", err)
		ctx.String(http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("save done | total number of records : %d", len(configured_week_time_table_records))

	//////////

	ctx.String(http.StatusOK, "week time table preference added")
}
