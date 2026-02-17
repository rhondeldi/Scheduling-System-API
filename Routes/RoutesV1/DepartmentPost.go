package RoutesV1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
	"golang.org/x/crypto/bcrypt"
)

/*
POST:

	"/department_add"
*/
func PostDepartment(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	add_department := Departments.Department{}

	if err := ctx.BindJSON(&add_department); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the department to be added")
		return
	}

	/////////////////////// hash the raw password ///////////////////////

	raw_passwrd_byte := []byte(add_department.SaltedHashedPassword)

	hash, hashErr := bcrypt.GenerateFromPassword(raw_passwrd_byte, bcrypt.DefaultCost)

	if hashErr != nil {
		fmt.Println(hashErr)
		ctx.String(http.StatusInternalServerError, "we're unable to process your request right now")
		return
	}

	bcrypt_hashed_passwrd_string := string(hash)

	/////////////////////// save the user ///////////////////////

	Utils.PrettyPrint(add_department)

	add_department.SaltedHashedPassword = bcrypt_hashed_passwrd_string

	Utils.PrettyPrint(add_department)

	err_create_department := RouteGlobals.ResourcesPersistence.WriterService.CreateDepartment(add_department)

	if err_create_department != nil {
		log.Print(err_create_department)
		ctx.String(http.StatusBadRequest, "we are unable to properly add the department")
		return
	}

	ctx.String(http.StatusOK, "department added successfully")
}
