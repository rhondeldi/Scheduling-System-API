package RouteGlobals

import (
	"errors"
	"fmt"
	"log"
	"slices"
	"sync"

	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
)

type SchedGenStatusType string

const (
	SchedGenStatusSuccess       SchedGenStatusType = "success"
	SchedGenStatusFailed        SchedGenStatusType = "failed"
	SchedGenStatusInternalError SchedGenStatusType = "internal server error"
	SchedGenStatusNotStarted    SchedGenStatusType = "not started"
	SchedGenStatusOnQueue       SchedGenStatusType = "on queue"
	SchedGenStatusInProgress    SchedGenStatusType = "in progress"
)

type SchedGenResult struct {
	Status                       SchedGenStatusType `json:"Status"` // values: "success", "failed", "internal server error", "not started", "on queue", "in progress"
	Message                      string             `json:"Message"`
	FitnessProgressionDepartment []float64          `json:"FitnessProgressionDepartment,omitempty"`
	FitnessProgressionUniversity []float64          `json:"FitnessProgressionUniversity,omitempty"`
}

var rw_map_mutex sync.RWMutex
var department_id_to_sched_gen_last_result map[DeptSchedGenKey]SchedGenResult

type DeptSchedGenKey struct {
	DepartmentID uint16
	Semester     int
}

func SetDeptSchedGenResult(key DeptSchedGenKey, value SchedGenResult) {
	rw_map_mutex.Lock()
	defer rw_map_mutex.Unlock()

	if department_id_to_sched_gen_last_result == nil {
		department_id_to_sched_gen_last_result = make(map[DeptSchedGenKey]SchedGenResult)
	}

	department_id_to_sched_gen_last_result[key] = value
}

func GetDepartSchedGenResult(key DeptSchedGenKey) SchedGenResult {
	rw_map_mutex.RLock()
	defer rw_map_mutex.RUnlock()

	if department_id_to_sched_gen_last_result == nil {
		department_id_to_sched_gen_last_result = make(map[DeptSchedGenKey]SchedGenResult)
	}

	departments, err_read_departments := ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_departments != nil {
		log.Print("GetDepartSchedGenResult: we're unable to retrieve the departments, caused by ", err_read_departments)
		return SchedGenResult{
			Status:  SchedGenStatusInternalError,
			Message: "something wrong happened, we're unable to retrieve the departments while processing the queue",
		}
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	last_sched_gen_result, has_key := department_id_to_sched_gen_last_result[key]

	if !has_key {

		msg := fmt.Sprintf(
			"no recent schedule generation in %s %s",
			dept_id_to_department[key.DepartmentID].Name,
			Curriculum.SEMESTER_INDEX_NAME[key.Semester],
		)

		return SchedGenResult{
			Status:  SchedGenStatusNotStarted,
			Message: msg,
		}
	}

	return last_sched_gen_result
}

var rw_queue_mutex sync.RWMutex
var departments_schedule_generation_queue []DeptSchedGenKey

// initialize department generation request count.
func InitDeptSchedGenQueue() {
	if departments_schedule_generation_queue == nil {
		log.Print("department to encode requests initialized")
		departments_schedule_generation_queue = make([]DeptSchedGenKey, 0)
	} else {
		log.Print("department to encode requests is already initialized")
	}
}

// return true if the department was added to the schedule generation task queue.
//
// return false if the department was already in the schedule generation task queue.
func PushNewDeptToDeptSchedGenQueue(department_id_and_semester DeptSchedGenKey) bool {
	rw_queue_mutex.RLock()
	defer rw_queue_mutex.RUnlock()

	if slices.Contains(departments_schedule_generation_queue, department_id_and_semester) {
		return false
	}

	for _, q := range departments_schedule_generation_queue {
		if q == department_id_and_semester {
			return false
		}
	}

	departments_schedule_generation_queue = append(departments_schedule_generation_queue, department_id_and_semester)
	return true
}

// returns only one department id mapped to a true boolean value.
//
// returns an error if the queue is empty or uninitialized.
func PopDepartmentToEncodeFromSchedGenQueue() (map[uint16]bool, int, error) {
	rw_queue_mutex.Lock()
	defer rw_queue_mutex.Unlock()

	if departments_schedule_generation_queue == nil {
		return nil, -1, errors.New("the current schedule generation queue is uninitialized")
	}

	if len(departments_schedule_generation_queue) == 0 {
		return nil, -1, errors.New("the current schedule generation queue is empty")
	}

	first_dept_sem := departments_schedule_generation_queue[0]

	if len(departments_schedule_generation_queue) > 1 {
		departments_schedule_generation_queue = departments_schedule_generation_queue[1:]
	} else {
		departments_schedule_generation_queue = make([]DeptSchedGenKey, 0)
	}

	department_to_encode := make(map[uint16]bool)
	department_to_encode[first_dept_sem.DepartmentID] = true

	return department_to_encode, first_dept_sem.Semester, nil
}
