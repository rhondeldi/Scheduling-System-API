package StorageResources

import (
	"encoding/json"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonReader) ReadAllSubjects() ([]Curriculum.Subject, error) {

	SubjectMutex.Lock()
	defer SubjectMutex.Unlock()

	return json_read_all_subjects()
}

func json_read_all_subjects() ([]Curriculum.Subject, error) {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return nil, err_project_root
	}

	subjects_json_file := path.Join(project_root, "scheduling-system-temporary-data", "all-subjects.json")
	subjects_byte_data, err := os.ReadFile(subjects_json_file)

	if err != nil {
		return nil, err
	}

	subjects := []Curriculum.Subject{}

	err = json.Unmarshal(subjects_byte_data, &subjects)

	if err != nil {
		return nil, err
	}

	sort.Slice(subjects, func(i, j int) bool {
		return subjects[i].ID < subjects[j].ID
	})

	return subjects, nil
}

// return a nil subject if subject does not exist
func (s *JsonReader) ReadSubject(subject_id uint16) (*Curriculum.Subject, error) {

	SubjectMutex.Lock()
	defer SubjectMutex.Unlock()

	subjects, err := json_read_all_subjects()

	if err != nil {
		return nil, err
	}

	for _, subject := range subjects {
		if subject.ID == subject_id {
			return &subject, nil
		}
	}

	return nil, nil
}
