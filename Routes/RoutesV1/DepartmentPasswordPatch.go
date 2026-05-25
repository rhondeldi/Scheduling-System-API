package RoutesV1

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"golang.org/x/crypto/bcrypt"
)

type DepartmentPasswordUpdateForm struct {
	DepartmentID    uint16 `json:"DepartmentID"`
	CurrentPassword string `json:"CurrentPassword"`
	NewPassword     string `json:"NewPassword"`
	ConfirmPassword string `json:"ConfirmPassword"`
}

func PatchDepartmentPassword(ctx *gin.Context) {
	if isSuccess := Auth.IsAuthSuccess(ctx); !isSuccess {
		return
	}

	form := DepartmentPasswordUpdateForm{}
	if err := ctx.BindJSON(&form); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the department password update request")
		return
	}

	if isAllowed := Auth.IsDepartmentAllowed(ctx, form.DepartmentID); !isAllowed {
		return
	}

	if form.DepartmentID == 0 {
		ctx.String(http.StatusBadRequest, "department account is required")
		return
	}

	if strings.TrimSpace(form.CurrentPassword) == "" || strings.TrimSpace(form.NewPassword) == "" || strings.TrimSpace(form.ConfirmPassword) == "" {
		ctx.String(http.StatusUnprocessableEntity, "current password, new password, and confirm password are required")
		return
	}

	if form.NewPassword != form.ConfirmPassword {
		ctx.String(http.StatusUnprocessableEntity, "passwords do not match")
		return
	}

	if len(form.NewPassword) < 8 {
		ctx.String(http.StatusUnprocessableEntity, "password length should be equal or above 8 characters")
		return
	}

	department, errReadDepartment := RouteGlobals.ResourcesPersistence.ReaderService.ReadDepartment(form.DepartmentID)
	if errReadDepartment != nil {
		log.Print("PatchDepartmentPassword: unable to retrieve department information")
		ctx.String(http.StatusInternalServerError, "we're unable to retrieve the department information")
		return
	}

	if len(os.Getenv("MASTER_KEY")) >= 32 {
		isEqual := subtle.ConstantTimeCompare(
			[]byte(os.Getenv("MASTER_KEY")),
			[]byte(form.CurrentPassword),
		) == 1

		if !isEqual {
			ctx.String(http.StatusUnauthorized, "current password is incorrect")
			return
		}
	} else if err := bcrypt.CompareHashAndPassword([]byte(department.SaltedHashedPassword), []byte(form.CurrentPassword)); err != nil {
		ctx.String(http.StatusUnauthorized, "current password is incorrect")
		return
	}

	hash, hashErr := bcrypt.GenerateFromPassword([]byte(form.NewPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		fmt.Println(hashErr)
		ctx.String(http.StatusInternalServerError, "we're unable to process your request right now")
		return
	}

	updateDepartment := Departments.Department{
		DepartmentID:         department.DepartmentID,
		Code:                 department.Code,
		Name:                 department.Name,
		SaltedHashedPassword: string(hash),
	}

	if err := RouteGlobals.ResourcesPersistence.WriterService.UpdateDepartment(updateDepartment); err != nil {
		log.Print(err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "department password updated successfully")
}
