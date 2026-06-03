package StorageResources

import (
	"encoding/json"
	"os"
	"path"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonReader) ReadAsyncScheduleRecords(department_id uint16, semester int) ([]Schedule.AsyncScheduleRecord, error) {
	AsyncScheduleMutex.Lock()
	defer AsyncScheduleMutex.Unlock()

	records, err := json_read_all_async_schedule_records()
	if err != nil {
		return nil, err
	}

	filtered := make([]Schedule.AsyncScheduleRecord, 0)
	for _, record := range records {
		if record.DepartmentID != department_id {
			continue
		}
		if record.Semester != semester {
			continue
		}
		filtered = append(filtered, record)
	}

	sortAsyncScheduleRecords(filtered)
	return filtered, nil
}

func json_read_all_async_schedule_records() ([]Schedule.AsyncScheduleRecord, error) {
	project_root, err_project_root := Utils.FindProjectRoot()
	if err_project_root != nil {
		return nil, err_project_root
	}

	records_json_file := path.Join(project_root, "scheduling-system-temporary-data", "async-schedule-records.json")
	records_byte_data, err := os.ReadFile(records_json_file)
	if err != nil {
		if os.IsNotExist(err) {
			return make([]Schedule.AsyncScheduleRecord, 0), nil
		}
		return nil, err
	}

	records := make([]Schedule.AsyncScheduleRecord, 0)
	if len(records_byte_data) == 0 {
		return records, nil
	}

	if err := json.Unmarshal(records_byte_data, &records); err != nil {
		return nil, err
	}

	sortAsyncScheduleRecords(records)
	return records, nil
}
