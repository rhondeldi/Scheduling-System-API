package RoutesV1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"golang.org/x/crypto/bcrypt"
)

/*
PATCH:

	"/department_update"
*/
func PatchDepartment(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	update_department := Departments.Department{}

	if err := ctx.BindJSON(&update_department); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the department updated data")
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, update_department.DepartmentID); !is_allowed {
		return
	}

	old_department, err_read_department := RouteGlobals.ResourcesPersistence.ReaderService.ReadDepartment(update_department.DepartmentID)

	if err_read_department != nil {
		log.Print("PatchDepartment: we're unable to retrieve the old curriculum information for comparison")
		ctx.String(http.StatusInternalServerError, "we're unable to retrieve the old curriculum information for comparison")
		return
	}

	if len(update_department.SaltedHashedPassword) != 0 {

		/////////////////////// hash the raw password ///////////////////////

		if len(update_department.SaltedHashedPassword) >= 8 {
			raw_passwrd_byte := []byte(update_department.SaltedHashedPassword)

			hash, hashErr := bcrypt.GenerateFromPassword(raw_passwrd_byte, bcrypt.DefaultCost)

			if hashErr != nil {
				fmt.Println(hashErr)
				ctx.String(http.StatusInternalServerError, "we're unable to process your request right now")
				return
			}

			bcrypt_hashed_passwrd_string := string(hash)

			update_department.SaltedHashedPassword = bcrypt_hashed_passwrd_string
		} else {
			ctx.String(http.StatusUnprocessableEntity, "password length should be equal or above 8 characters")
			return
		}
	} else {
		update_department.SaltedHashedPassword = old_department.SaltedHashedPassword
	}

	/////////////////////// save the user ///////////////////////

	err := RouteGlobals.ResourcesPersistence.WriterService.UpdateDepartment(update_department)

	if err != nil {
		log.Print(err)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}

	ctx.String(http.StatusOK, "department updated successfully")
}
