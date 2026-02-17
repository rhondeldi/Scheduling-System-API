package StorageSchedule

import (
	"sync"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

var UniSchedPersistenceMutex sync.Mutex

type loadRepository interface {
	LoadSchedules(semester int) (Schedule.UniTimeTables, error)
}

type saveRepository interface {
	SaveSchedules(university_schedule Schedule.UniTimeTables, semester int) error
	DeleteSchedules(semester int) error
}

type Persistence struct {
	SaveService saveRepository
	LoadService loadRepository
}
