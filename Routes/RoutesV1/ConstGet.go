package RoutesV1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

/*
GET:

	"/const"
*/
func GetConst(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"time_slot_bytes":    Schedule.TIME_SLOT_BYTE_SIZE,
		"daily_school_hours": Const.N_DAILY_SCHOOL_HOURS,
		"time_slot_per_hour": Const.N_HOUR_TIME_SLOTS,
		"weekly_school_days": Const.N_WEEKLY_SCHOOL_DAYS,
		"daily_time_slots":   Const.N_DAILY_TIME_SLOTS,
		"weekly_time_slots":  Const.N_WEEKLY_TIME_SLOTS,
	})
}
