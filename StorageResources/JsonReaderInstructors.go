package StorageResources

import (
	"encoding/json"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonReader) ReadAllInstructors() ([]Instructors.Instructor, error) {

	instructors_with_time_string, err := s.ReadAllInstructorsWithTimeString()

	if err != nil {
		return nil, err
	}

	instructors := make([]Instructors.Instructor, 0)

	for _, instructor_with_time_str := range instructors_with_time_string {
		instructor := Instructors.Instructor{
			InstructorID:  instructor_with_time_str.InstructorID,
			DepartmentID:  instructor_with_time_str.DepartmentID,
			FirstName:     instructor_with_time_str.FirstName,
			MiddleInitial: instructor_with_time_str.MiddleInitial,
			LastName:      instructor_with_time_str.LastName,
			EmploymentType: instructor_with_time_str.EmploymentType,
			MaxUnits:       instructor_with_time_str.MaxUnits,
			DesignatedSubjectIDs: instructor_with_time_str.DesignatedSubjectIDs,
		}

		instructor.Time.StringParse(instructor_with_time_str.Time)

		instructors = append(instructors, instructor)
	}

	return instructors, nil
}

func (s *JsonReader) ReadDepartmentInstructors(department_id int) ([]Instructors.Instructor, error) {

	instructors_with_time_string, err := s.ReadAllInstructorsWithTimeString()

	if err != nil {
		return nil, err
	}

	department_instructors := make([]Instructors.Instructor, 0)

	for _, instructor_with_time_str := range instructors_with_time_string {
		if instructor_with_time_str.DepartmentID != uint16(department_id) {
			continue
		}

		instructor := Instructors.Instructor{
			InstructorID:  instructor_with_time_str.InstructorID,
			DepartmentID:  instructor_with_time_str.DepartmentID,
			FirstName:     instructor_with_time_str.FirstName,
			MiddleInitial: instructor_with_time_str.MiddleInitial,
			LastName:      instructor_with_time_str.LastName,
			EmploymentType: instructor_with_time_str.EmploymentType,
			MaxUnits:       instructor_with_time_str.MaxUnits,
			DesignatedSubjectIDs: instructor_with_time_str.DesignatedSubjectIDs,
		}

		instructor.Time.StringParse(instructor_with_time_str.Time)

		department_instructors = append(department_instructors, instructor)
	}

	return department_instructors, nil
}

func (s *JsonReader) ReadAllInstructorsWithTimeString() ([]Instructors.InstructorWithTimeString, error) {

	InstructorMutex.Lock()
	defer InstructorMutex.Unlock()

	return json_read_all_instructors_with_time_string()
}

func json_read_all_instructors_with_time_string() ([]Instructors.InstructorWithTimeString, error) {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return nil, err_project_root
	}

	instructors_json_file := path.Join(project_root, "scheduling-system-temporary-data", "instructors.json")
	instructors_byte_data, err := os.ReadFile(instructors_json_file)

	if err != nil {
		return nil, err
	}

	instructors := make([]Instructors.InstructorWithTimeString, 0)
	err = json.Unmarshal(instructors_byte_data, &instructors)

	if err != nil {
		return nil, err
	}

	sort.Slice(instructors, func(i, j int) bool {
		return instructors[i].InstructorID < instructors[j].InstructorID
	})

	return instructors, nil
}

// return a nil instructor if instructor does not exist
func (s *JsonReader) ReadInstructor(instructor_id uint16) (*Instructors.Instructor, error) {

	instructors, err := s.ReadAllInstructors()

	if err != nil {
		return nil, err
	}

	for _, instructor := range instructors {
		if instructor.InstructorID == instructor_id {
			return &instructor, nil
		}
	}

	return nil, nil
}
