package StorageResources

import (
	"sync"

	AdminResource "github.com/mrdcvlsc/scheduling-system-backend/Resources/Admin"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

var SubjectMutex sync.Mutex
var CurriculumsMutex sync.Mutex
var DepartmentMutex sync.Mutex
var InstructorMutex sync.Mutex
var RoomMutex sync.Mutex
var AsyncScheduleMutex sync.Mutex

type readerRepository interface {
	ReadAdminCredentials() (*AdminResource.AdminCredentials, error)

	// the order of curriculums returned by this method is always sorted by curriculum ID.
	ReadAllCurriculum() ([]Curriculum.Curriculum, error)
	ReadCurriculum(curriculum_id uint16) (*Curriculum.Curriculum, error)

	ReadAllSubjects() ([]Curriculum.Subject, error)
	ReadSubject(subject_id uint16) (*Curriculum.Subject, error)

	ReadAllDepartments() ([]Departments.Department, error)
	ReadDepartment(department_id uint16) (*Departments.Department, error)

	ReadAllInstructors() ([]Instructors.Instructor, error)
	ReadInstructor(instructor_id uint16) (*Instructors.Instructor, error)
	ReadDepartmentInstructors(department_id int) ([]Instructors.Instructor, error)

	ReadAllRooms() ([]Rooms.Room, error)
	ReadRoom(room_id uint16) (*Rooms.Room, error)

	ReadAsyncScheduleRecords(department_id uint16, semester int) ([]Schedule.AsyncScheduleRecord, error)
}

type writerRepository interface {
	UpsertAdminCredentials(credentials AdminResource.AdminCredentials) error

	CreateDepartment(new_department Departments.Department) error
	UpdateDepartment(department_to_update Departments.Department) error
	DeleteDepartment(department_id uint16) error

	CreateSubject(new_subject Curriculum.Subject) error
	UpdateSubject(subject_to_update Curriculum.Subject) error
	DeleteSubject(subject_id uint16) error

	CreateCurriculum(new_curriculum Curriculum.Curriculum) (uint16, error) // returns the CurriculumID and error result of the new curriculum
	UpdateCurriculum(updated_curriculum Curriculum.Curriculum) error
	DeleteCurriculum(curriculum_id uint16) error

	CreateInstructor(new_instructor Instructors.Instructor) error
	UpdateInstructor(instructor_to_update Instructors.Instructor) error
	DeleteInstructor(instructor_id uint16) error

	CreateRoom(new_room Rooms.Room) error
	UpdateRoom(room_to_update Rooms.Room) error
	DeleteRoom(room_id uint16) error

	ReplaceAsyncScheduleRecords(department_id uint16, semester int, records []Schedule.AsyncScheduleRecord) error
	DeleteAsyncScheduleRecords(department_id uint16, semester int) error
}

type Persistence struct {
	ReaderService readerRepository
	WriterService writerRepository
}
