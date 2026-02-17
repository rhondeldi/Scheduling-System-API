package StorageResources

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) CreateInstructor(new_instructor Instructors.Instructor) error {

	if new_instructor.InstructorID != 0 {
		return errors.New("cannot create a new instructor with a non zero instructor ID because it will overwrite an instructor")
	}

	InstructorMutex.Lock()
	defer InstructorMutex.Unlock()

	instructors_with_time_str, err_read := json_read_all_instructors_with_time_string()

	if err_read != nil {
		return err_read
	}

	new_id_num := uint16(1)

	if len(instructors_with_time_str) > 0 {
		new_id_num = instructors_with_time_str[len(instructors_with_time_str)-1].InstructorID + 1
	}

	instructors_with_time_str = append(instructors_with_time_str, Instructors.InstructorWithTimeString{
		InstructorID:  new_id_num,
		DepartmentID:  new_instructor.DepartmentID,
		FirstName:     new_instructor.FirstName,
		MiddleInitial: new_instructor.MiddleInitial,
		LastName:      new_instructor.LastName,
		Time:          new_instructor.Time.Stringify(),
	})

	err_save_instructors := json_save_all_instructors_with_time_string(instructors_with_time_str)

	if err_save_instructors != nil {
		return err_save_instructors
	}

	return nil
}

func (s *JsonWriter) UpdateInstructor(instructor_to_update Instructors.Instructor) error {

	if instructor_to_update.InstructorID == 0 {
		return errors.New("parameter argument missing invalid instructor ID")
	}

	InstructorMutex.Lock()
	defer InstructorMutex.Unlock()

	instructors_with_time_str, err_read := json_read_all_instructors_with_time_string()

	if err_read != nil {
		return err_read
	}

	has_id := false
	to_update_idx := -1

	for idx, instructor_w_t_str := range instructors_with_time_str {
		if instructor_w_t_str.InstructorID == instructor_to_update.InstructorID {
			has_id = true
			to_update_idx = idx
			break
		}
	}

	if !has_id {
		return errors.New("instructor to update does not exist in the json file")
	}

	instructors_with_time_str[to_update_idx] = Instructors.InstructorWithTimeString{
		InstructorID:  instructor_to_update.InstructorID,
		DepartmentID:  instructor_to_update.DepartmentID,
		FirstName:     instructor_to_update.FirstName,
		MiddleInitial: instructor_to_update.MiddleInitial,
		LastName:      instructor_to_update.LastName,
		Time:          instructor_to_update.Time.Stringify(),
	}

	err_save_instructors := json_save_all_instructors_with_time_string(instructors_with_time_str)

	if err_save_instructors != nil {
		return err_save_instructors
	}

	return nil
}

func (s *JsonWriter) DeleteInstructor(instructor_id uint16) error {

	if instructor_id == 0 {
		return errors.New("parameter argument missing invalid instructor ID")
	}

	InstructorMutex.Lock()
	defer InstructorMutex.Unlock()

	instructors_with_time_str, err_read := json_read_all_instructors_with_time_string()

	if err_read != nil {
		return err_read
	}

	instructors_with_time_string_deleted := make([]Instructors.InstructorWithTimeString, 0)

	has_id := false

	for _, instructor_w_t_str := range instructors_with_time_str {
		if instructor_w_t_str.InstructorID == instructor_id {
			has_id = true
		} else {
			instructors_with_time_string_deleted = append(instructors_with_time_string_deleted, instructor_w_t_str)
		}
	}

	if !has_id {
		return errors.New("instructor to delete does not exist in the json file")
	}

	err_save_instructors := json_save_all_instructors_with_time_string(instructors_with_time_string_deleted)

	if err_save_instructors != nil {
		return err_save_instructors
	}

	return nil
}

func json_save_all_instructors_with_time_string(instructors []Instructors.InstructorWithTimeString) error {
	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	instructors_json_file := path.Join(project_root, "scheduling-system-temporary-data", "instructors.json")

	sort.Slice(instructors, func(i, j int) bool {
		return instructors[i].InstructorID < instructors[j].InstructorID
	})

	instructors_byte_data, err := json.MarshalIndent(instructors, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(instructors_json_file, instructors_byte_data, 0644); err != nil {
		return err
	}

	return nil
}
