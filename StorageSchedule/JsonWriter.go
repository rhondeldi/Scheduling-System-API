package StorageSchedule

import (
	"fmt"
	"os"
	"path"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type JsonWriter struct{}

func (s *JsonWriter) SaveSchedules(university_schedule Schedule.UniTimeTables, semester int) error {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	var saved_file string

	for semester_idx := range len(Curriculum.SEMESTER_INDEX_NAME) {
		if semester_idx == semester {
			saved_file = path.Join(project_root, "scheduling-system-temporary-data", fmt.Sprintf("univ-sem-%d.sched", semester_idx+1))
			break
		}
	}

	UniSchedPersistenceMutex.Lock()
	defer UniSchedPersistenceMutex.Unlock()

	Utils.SaveToBinFile(saved_file, Schedule.SerializeUniversitySchedule(university_schedule))

	return nil
}

func (s *JsonWriter) DeleteSchedules(semester int) error {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	var file_to_delete string

	for semester_idx := range len(Curriculum.SEMESTER_INDEX_NAME) {
		if semester_idx == semester {
			file_to_delete = path.Join(project_root, "scheduling-system-temporary-data", fmt.Sprintf("univ-sem-%d.sched", semester_idx+1))
			break
		}
	}

	if file_to_delete == "" {
		return fmt.Errorf("no file found for semester %d", semester)
	}

	UniSchedPersistenceMutex.Lock()
	defer UniSchedPersistenceMutex.Unlock()

	err := os.Remove(file_to_delete)

	if err != nil {
		return err
	}

	return nil
}
