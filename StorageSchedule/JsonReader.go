package StorageSchedule

import (
	"fmt"
	"path"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type JsonReader struct{}

func (s *JsonReader) LoadSchedules(semester int) (Schedule.UniTimeTables, error) {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return nil, err_project_root
	}

	var saved_file string

	for semester_idx := range Curriculum.SUPPORTED_SEMESTERS {
		if semester_idx == semester {
			saved_file = path.Join(project_root, "scheduling-system-temporary-data", fmt.Sprintf("univ-sem-%d.sched", semester_idx+1))
			break
		}
	}

	UniSchedPersistenceMutex.Lock()
	defer UniSchedPersistenceMutex.Unlock()

	read_bytes, err_read_from_bin_file := Utils.ReadFromBinFile(saved_file)

	if err_read_from_bin_file != nil {
		return nil, err_read_from_bin_file
	}

	deserialize_data := Schedule.DeserializeUniversitySchedule(read_bytes)

	return deserialize_data, nil
}
