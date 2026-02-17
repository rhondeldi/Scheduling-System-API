package RoutesV1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
)

/*
PATCH:

	"/subject_update"
*/
func PatchSubject(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	update_subject := Curriculum.Subject{}

	if err := ctx.BindJSON(&update_subject); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the subject updated data")
		return
	}

	err := RouteGlobals.ResourcesPersistence.WriterService.UpdateSubject(update_subject)

	if err != nil {
		log.Print(err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "subject updated successfully")
}
