package StorageResources

import (
	"encoding/json"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) ReplaceAsyncScheduleRecords(department_id uint16, semester int, records []Schedule.AsyncScheduleRecord) error {
	AsyncScheduleMutex.Lock()
	defer AsyncScheduleMutex.Unlock()

	all_records, err_read := json_read_all_async_schedule_records()
	if err_read != nil {
		return err_read
	}

	next_records := make([]Schedule.AsyncScheduleRecord, 0, len(all_records)+len(records))
	for _, record := range all_records {
		if record.DepartmentID == department_id && record.Semester == semester {
			continue
		}
		next_records = append(next_records, record)
	}

	next_records = append(next_records, records...)

	return json_save_all_async_schedule_records(next_records)
}

func (s *JsonWriter) DeleteAsyncScheduleRecords(department_id uint16, semester int) error {
	return s.ReplaceAsyncScheduleRecords(department_id, semester, nil)
}

func json_save_all_async_schedule_records(records []Schedule.AsyncScheduleRecord) error {
	project_root, err_project_root := Utils.FindProjectRoot()
	if err_project_root != nil {
		return err_project_root
	}

	records_json_file := path.Join(project_root, "scheduling-system-temporary-data", "async-schedule-records.json")
	sortAsyncScheduleRecords(records)

	records_byte_data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(records_json_file, records_byte_data, 0644)
}

func sortAsyncScheduleRecords(records []Schedule.AsyncScheduleRecord) {
	if len(records) <= 1 {
		return
	}

	// deterministic ordering improves diffs and debugging
	sort.Slice(records, func(i, j int) bool {
		if records[i].DepartmentID != records[j].DepartmentID {
			return records[i].DepartmentID < records[j].DepartmentID
		}
		if records[i].Semester != records[j].Semester {
			return records[i].Semester < records[j].Semester
		}
		if records[i].InstructorID != records[j].InstructorID {
			return records[i].InstructorID < records[j].InstructorID
		}
		if records[i].SubjectID != records[j].SubjectID {
			return records[i].SubjectID < records[j].SubjectID
		}
		return records[i].SectionUSI < records[j].SectionUSI
	})
}
