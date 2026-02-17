package StorageResources

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) CreateDepartment(new_department Departments.Department) error {

	if new_department.DepartmentID != 0 {
		return errors.New("cannot create a newdepartment with a non zero department ID because that would overwrite a department")
	}

	DepartmentMutex.Lock()
	defer DepartmentMutex.Unlock()

	all_departments, err_read := json_read_all_departments()

	if err_read != nil {
		return err_read
	}

	new_id_num := uint16(1)

	if len(all_departments) > 0 {
		new_id_num = all_departments[len(all_departments)-1].DepartmentID + 1
	}

	all_departments = append(all_departments, Departments.Department{
		DepartmentID:         new_id_num,
		Code:                 new_department.Code,
		Name:                 new_department.Name,
		SaltedHashedPassword: new_department.SaltedHashedPassword,
	})

	err_save_departments := json_save_all_departments(all_departments)

	if err_save_departments != nil {
		return err_save_departments
	}

	return nil
}

func (s *JsonWriter) UpdateDepartment(department_to_update Departments.Department) error {

	if department_to_update.DepartmentID == 0 {
		return errors.New("parameter argument missing invalid department ID")
	}

	DepartmentMutex.Lock()
	defer DepartmentMutex.Unlock()

	all_departments, err_read := json_read_all_departments()

	if err_read != nil {
		return err_read
	}

	has_id := false
	to_update_idx := -1

	for idx, department := range all_departments {
		if department.DepartmentID == department_to_update.DepartmentID {
			has_id = true
			to_update_idx = idx
			break
		}
	}

	if !has_id {
		return errors.New("department to update does not exist in the json file")
	}

	all_departments[to_update_idx] = department_to_update

	err_save_departments := json_save_all_departments(all_departments)

	if err_save_departments != nil {
		return err_save_departments
	}

	return nil
}

func (s *JsonWriter) DeleteDepartment(department_id uint16) error {

	if department_id == 0 {
		return errors.New("parameter argument missing invalid department ID")
	}

	DepartmentMutex.Lock()
	defer DepartmentMutex.Unlock()

	all_departments, err_read := json_read_all_departments()

	if err_read != nil {
		return err_read
	}

	other_departments := make([]Departments.Department, 0)

	has_id := false

	for _, department := range all_departments {
		if department.DepartmentID == department_id {
			has_id = true
		} else {
			other_departments = append(other_departments, department)
		}
	}

	if !has_id {
		return errors.New("department to delete does not exist in the json file")
	}

	err_save_departments := json_save_all_departments(other_departments)

	if err_save_departments != nil {
		return err_save_departments
	}

	return nil
}

func json_save_all_departments(departments []Departments.Department) error {
	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	departments_json_file := path.Join(project_root, "scheduling-system-temporary-data", "departments.json")

	sort.Slice(departments, func(i, j int) bool {
		return departments[i].DepartmentID < departments[j].DepartmentID
	})

	departments_byte_data, err := json.MarshalIndent(departments, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(departments_json_file, departments_byte_data, 0644); err != nil {
		return err
	}

	return nil
}
