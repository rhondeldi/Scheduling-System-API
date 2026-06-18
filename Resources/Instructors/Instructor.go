package Instructors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

// instructor employment divisions.
const (
	EMPLOYMENT_TYPE_REGULAR   = "regular"
	EMPLOYMENT_TYPE_PART_TIME = "part-time"
)

// NormalizedEmploymentType returns a canonical employment type, defaulting to
// regular when the value is empty or unrecognized.
func (instructor *Instructor) NormalizedEmploymentType() string {
	switch strings.ToLower(strings.TrimSpace(instructor.EmploymentType)) {
	case EMPLOYMENT_TYPE_PART_TIME:
		return EMPLOYMENT_TYPE_PART_TIME
	default:
		return EMPLOYMENT_TYPE_REGULAR
	}
}

// EffectiveMaxUnits returns the weekly teaching unit cap that applies to this
// instructor. Regular instructors are always capped at the fixed regular
// maximum (not configurable per-instructor).
//
// Part-time instructors are capped by their weekly availability instead: roughly
// one unit per available teaching hour (see InstructorTimeSlotBitMap.
// CountAvailableHours), so the less available a part-timer is, the fewer units
// they can carry. This availability cap is always kept below a full regular
// load. An explicit per-instructor MaxUnits, if set, can only tighten the cap
// further — it can never raise it above what availability allows.
func (instructor *Instructor) EffectiveMaxUnits() uint8 {
	if instructor.NormalizedEmploymentType() == EMPLOYMENT_TYPE_REGULAR {
		return Const.REGULAR_INSTRUCTOR_MAX_UNITS
	}

	cap := instructor.Time.CountAvailableHours()
	if cap >= Const.REGULAR_INSTRUCTOR_MAX_UNITS {
		cap = Const.REGULAR_INSTRUCTOR_MAX_UNITS - 1
	}

	if instructor.MaxUnits != 0 && instructor.MaxUnits < cap {
		cap = instructor.MaxUnits
	}

	return cap
}

// Validate trims the instructor's name fields and ensures the required
// fields are not empty, and normalizes the employment division / unit cap.
// It returns an error describing the first problem found, or nil if the
// instructor is valid.
func (instructor *Instructor) Validate() error {
	instructor.FirstName = strings.TrimSpace(instructor.FirstName)
	instructor.LastName = strings.TrimSpace(instructor.LastName)
	instructor.MiddleInitial = strings.TrimSpace(instructor.MiddleInitial)

	if instructor.FirstName == "" {
		return errors.New("instructor first name cannot be empty")
	}

	if instructor.LastName == "" {
		return errors.New("instructor last name cannot be empty")
	}

	instructor.EmploymentType = instructor.NormalizedEmploymentType()

	if instructor.EmploymentType == EMPLOYMENT_TYPE_REGULAR {
		// regular instructors are fixed at the regular cap and carry no
		// per-instructor max.
		instructor.MaxUnits = Const.REGULAR_INSTRUCTOR_MAX_UNITS
	} else {
		// part-time caps are derived from availability (see EffectiveMaxUnits),
		// so a 0 MaxUnits is valid and simply means "auto from availability".
		// An explicit cap, when provided, only tightens that and must stay below
		// a full regular load.
		if instructor.MaxUnits >= Const.REGULAR_INSTRUCTOR_MAX_UNITS {
			return fmt.Errorf(
				"part-time instructor unit cap (%d) must be less than the regular cap of %d units",
				instructor.MaxUnits, Const.REGULAR_INSTRUCTOR_MAX_UNITS,
			)
		}
	}

	return nil
}

// type that will be use to read and save a single instructor's information from and to the database.
type Instructor struct {
	InstructorID  uint16 `json:"InstructorID" bson:"InstructorID"` // this should never be zero, zero means empty, none or nothing.
	DepartmentID  uint16 `json:"DepartmentID" bson:"DepartmentID"`
	FirstName     string `json:"FirstName" bson:"FirstName"`
	MiddleInitial string `json:"MiddleInitial" bson:"MiddleInitial"`
	LastName      string `json:"LastName" bson:"LastName"`

	// employment division: "regular" or "part-time". Empty is treated as regular.
	EmploymentType string `json:"EmploymentType,omitempty" bson:"EmploymentType,omitempty"`

	// weekly teaching unit cap. For regular instructors this is fixed at the
	// regular cap. For part-time instructors the cap is derived from their
	// availability; this field is then an OPTIONAL extra ceiling (0 = none) that
	// can only lower that derived value. Resolve the effective value through
	// EffectiveMaxUnits() rather than reading this directly.
	MaxUnits uint8 `json:"MaxUnits,omitempty" bson:"MaxUnits,omitempty"`

	// specialization subject IDs this instructor can handle.
	DesignatedSubjectIDs []uint16 `json:"DesignatedSubjectIDs,omitempty" bson:"DesignatedSubjectIDs,omitempty"`

	// the total number of assigned subjects to teach.
	AssignedSubjects int

	// the total teaching hours assigned for all assigned subjects.
	TotalTeachingHours float32

	// the total credit units assigned for all assigned subjects (in-memory only,
	// accumulated during schedule generation to enforce the unit cap).
	AssignedUnits uint16

	// determines if the instructor is available for a certain time slot during schedule generation.
	//
	// if value corresponding bit value for that time slot is 1, then the instructor is available.
	// if value corresponding bit value for that time slot is 0, then the instructor is NOT available.
	Time InstructorTimeSlotBitMap
}

type InstructorWithTimeString struct {
	InstructorID  uint16   `json:"InstructorID" bson:"InstructorID"`
	DepartmentID  uint16   `json:"DepartmentID" bson:"DepartmentID"`
	FirstName     string   `json:"FirstName" bson:"FirstName"`
	MiddleInitial string   `json:"MiddleInitial" bson:"MiddleInitial"`
	LastName      string   `json:"LastName" bson:"LastName"`
	EmploymentType string  `json:"EmploymentType,omitempty" bson:"EmploymentType,omitempty"`
	MaxUnits      uint8    `json:"MaxUnits,omitempty" bson:"MaxUnits,omitempty"`
	DesignatedSubjectIDs []uint16 `json:"DesignatedSubjectIDs,omitempty" bson:"DesignatedSubjectIDs,omitempty"`
	Time          []string `json:"Time" bson:"Time"`
}
