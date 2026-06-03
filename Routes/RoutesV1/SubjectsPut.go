package RoutesV1

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
)

// PUT /subjects/:id
func PutSubject(ctx *gin.Context) {
	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	subject_id, err_parse_id := strconv.Atoi(ctx.Param("id"))
	if err_parse_id != nil || subject_id <= 0 {
		ctx.String(http.StatusBadRequest, "invalid subject id path parameter")
		return
	}

	payload := SubjectUpsertPayload{}
	if err := ctx.BindJSON(&payload); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the subject updated data")
		return
	}

	payload.ID = uint16(subject_id)
	update_subject := buildSubjectFromPayload(payload)

	if err := normalizeAndValidateSubjectPayload(&update_subject); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	if err := RouteGlobals.ResourcesPersistence.WriterService.UpdateSubject(update_subject); err != nil {
		log.Print(err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "subject updated successfully")
}
