package RouteGlobals

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var IsGeneratingSchedule atomic.Bool
var RecentScheduleUpdate atomic.Uint64
var ReindexUniSchedMutex sync.Mutex

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type scheduleCache struct {
	rw_mutex          sync.RWMutex
	semester_schedule [Curriculum.SUPPORTED_SEMESTERS]Schedule.UniTimeTables
}

var schedule_cache *scheduleCache

/*
call this first on the main program, if not called panic will happen when calling the other two methods:

	GetCachedUniversitySchedule
	SetCachedUniversitySchedule
*/
func InitializeCachedUniversitySchedule() {
	if schedule_cache == nil {
		log.Print("cache for schedule initialized")
		schedule_cache = &scheduleCache{}
	} else {
		log.Print("cache for schedule is already initialized")
	}
}

/*
requires initial method call:

	InitializeCachedUniversitySchedule()

example usage:

	schedule, has_cache, err := RouteGlobals.GetCachedUniversitySchedule(semester)

	if err != nil {
		// error handling...
	}

	if has_cache {
		// do stuffs...
	}
*/
func GetCachedUniversitySchedule(semester int) (Schedule.UniTimeTables, bool, error) {

	if semester < 0 {
		return nil, false, errors.New("cached schedule semester index underflow")
	}

	if semester >= Curriculum.SUPPORTED_SEMESTERS {
		return nil, false, errors.New("cached schedule semester index overflow")
	}

	if schedule_cache == nil {
		return nil, false, errors.New("uninitialized scheduling cache")
	}

	schedule_cache.rw_mutex.RLock()
	defer schedule_cache.rw_mutex.RUnlock()

	if schedule_cache.semester_schedule[semester] != nil {
		if len(schedule_cache.semester_schedule[semester]) > 0 {
			return schedule_cache.semester_schedule[semester], true, nil
		}
	}

	return nil, false, nil
}

/*
requires initial method call:

	InitializeCachedUniversitySchedule()
*/
func SetCachedUniversitySchedule(semester int, university_schedule Schedule.UniTimeTables) error {

	if semester < 0 {
		return errors.New("cached schedule semester index underflow")
	}

	if semester >= Curriculum.SUPPORTED_SEMESTERS {
		return errors.New("cached schedule semester index overflow")
	}

	schedule_cache.rw_mutex.Lock()
	defer schedule_cache.rw_mutex.Unlock()

	schedule_cache.semester_schedule[semester] = university_schedule

	return nil
}

func ClearCachedUniversitySchedule() {
	for semester := range len(schedule_cache.semester_schedule) {
		schedule_cache.semester_schedule[semester] = nil
	}
}
