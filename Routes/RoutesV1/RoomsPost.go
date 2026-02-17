package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
)

/*
POST:

	"/room_add"
*/
func PostRoom(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	add_room := Rooms.Room{}

	if err := ctx.BindJSON(&add_room); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the room to be added")
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, add_room.DepartmentID); !is_allowed {
		return
	}

	err := RouteGlobals.ResourcesPersistence.WriterService.CreateRoom(add_room)

	if err != nil {
		log.Print("PostRoom : ", err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "room added successfully")
}
