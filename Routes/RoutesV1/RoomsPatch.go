package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

/*
PATCH:

	"/room_update"
*/
func PatchRoom(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	update_room := Rooms.Room{}

	if err := ctx.BindJSON(&update_room); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the room updated data")
		return
	}

	Utils.PrettyPrint(update_room)

	selected_room, err_read_room := RouteGlobals.ResourcesPersistence.ReaderService.ReadRoom(update_room.RoomID)

	if err_read_room != nil {
		ctx.String(http.StatusInternalServerError, "we're unable to find that room right now")
		return
	}

	if selected_room.DepartmentID != update_room.DepartmentID {
		if RouteGlobals.IsGeneratingSchedule.Load() {
			log.Print("PatchRoom: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
			ctx.String(http.StatusForbidden, "we're unable to move the room to other departments right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
			return
		}

		RouteGlobals.ReindexUniSchedMutex.Lock()
		defer RouteGlobals.ReindexUniSchedMutex.Unlock()
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, selected_room.DepartmentID); !is_allowed {
		return
	}

	err := RouteGlobals.ResourcesPersistence.WriterService.UpdateRoom(update_room)

	if err != nil {
		log.Print("PatchRoom : ", err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "room updated successfully")
}
