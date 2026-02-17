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
POST:

	"/subject_add"
*/
func PostSubject(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	add_subject := Curriculum.Subject{}

	if err := ctx.BindJSON(&add_subject); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the subject to be added")
		return
	}

	err := RouteGlobals.ResourcesPersistence.WriterService.CreateSubject(add_subject)

	if err != nil {
		log.Print(err)
		ctx.String(http.StatusBadRequest, "we are unable to properly add the subject")
		return
	}

	ctx.String(http.StatusOK, "subject added successfully")
}
