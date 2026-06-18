package Curriculum

import (
	"math"
	"strings"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

const LEC_HOURS_BIT_MASK uint32 = 0b11111111
const LAB_HOURS_BIT_MASK uint32 = 0b11111111 << 8
const FLAG_BIT_MASK uint32 = 0b1111111111111111 << 16

const SUBJECT_TYPE_LECTURE = "lecture"
const SUBJECT_TYPE_LABORATORY = "laboratory"

// use to read and write subject information from and to the database.
type Subject struct {
	// this should never be zero, zero means empty, none or nothing.
	ID       uint16 `json:"ID,omitempty" bson:"ID,omitempty"`
	Code     string `json:"Code" bson:"Code"`
	Name     string `json:"Name" bson:"Name"`
	LecHours uint8  `json:"LecHours" bson:"LecHours"`
	LabHours uint8  `json:"LabHours" bson:"LabHours"`

	// academic credit units of the subject, split into the lecture component and
	// the laboratory component. Distinct from contact hours (LecHours/LabHours).
	LecUnits uint8 `json:"LecUnits" bson:"LecUnits"`
	LabUnits uint8 `json:"LabUnits" bson:"LabUnits"`

	// total credit units = LecUnits + LabUnits. Server-computed on save and used
	// as the value counted toward an instructor's weekly unit cap during schedule
	// generation. Defaults to 0 for legacy subjects, in which case the unit cap is
	// effectively not enforced for them.
	Units uint8 `json:"Units" bson:"Units"`

	// lecture or laboratory
	SubjectType string `json:"SubjectType,omitempty" bson:"SubjectType,omitempty"`

	// async (self-study) hours that should not consume room slots in the weekly grid.
	AsynchronousHours float64 `json:"AsynchronousHours,omitempty" bson:"AsynchronousHours,omitempty"`

	// computed as total_hours - async_hours; kept for reference in APIs.
	SynchronousHours float64 `json:"SynchronousHours,omitempty" bson:"SynchronousHours,omitempty"`

	// optional explicit scheduling rule for special subjects.
	SaturdayOnly bool `json:"SaturdayOnly,omitempty" bson:"SaturdayOnly,omitempty"`

	// [15-bit unused][1-bit is gym boolean] - use to store other information about a subject, for now its only use case is to determine if a subject is a gym subject
	BitFlags uint16 `json:"BitFlags" bson:"BitFlags"`

	// if there are no designated instructor IDs here, the algorithm
	// will assign random instructors from the department.
	//
	// if there are some designated instructor IDs here, the algorithm
	// will immediately assign the instructor to the allocated subject
	// time slot, in the InstructorMonitor
	DesignatedInstructors []uint16 `json:"DesignatedInstructorsID,omitempty" bson:"DesignatedInstructorsID,omitempty"`
}

// [16-bit flags hrs][8-bit lab hrs][8-bit lec hrs]
func (s *Subject) Serialize() uint32 {
	return (uint32(s.BitFlags) << 16) | (uint32(s.LabHours) << 8) | uint32(s.LecHours)
}

// [16-bit flags hrs][8-bit lab hrs][8-bit lec hrs]
func (s *Subject) Deserialize(serialized_data uint32) {
	s.LecHours = uint8(serialized_data & LEC_HOURS_BIT_MASK)
	s.LabHours = uint8((serialized_data & LAB_HOURS_BIT_MASK) >> 8)
	s.BitFlags = uint16((serialized_data & FLAG_BIT_MASK) >> 16)
}

func (s *Subject) IsGymType() bool {
	return (s.BitFlags & 1) == 1
}

func (s Subject) TotalHours() float64 {
	return float64(s.LecHours) + float64(s.LabHours)
}

func (s Subject) NormalizedSubjectType() string {
	normalized := strings.ToLower(strings.TrimSpace(s.SubjectType))

	switch normalized {
	case SUBJECT_TYPE_LECTURE:
		return SUBJECT_TYPE_LECTURE
	case SUBJECT_TYPE_LABORATORY:
		return SUBJECT_TYPE_LABORATORY
	default:
		// Backward compatibility: infer from current hour shape if unset.
		if s.LabHours > 0 && s.LecHours == 0 {
			return SUBJECT_TYPE_LABORATORY
		}
		return SUBJECT_TYPE_LECTURE
	}
}

func (s Subject) IsLectureType() bool {
	return s.NormalizedSubjectType() == SUBJECT_TYPE_LECTURE
}

func (s Subject) IsLaboratoryType() bool {
	return s.NormalizedSubjectType() == SUBJECT_TYPE_LABORATORY
}

func (s Subject) EffectiveAsynchronousHours() float64 {
	if !s.IsLectureType() {
		return 0
	}

	if s.AsynchronousHours < 0 {
		return 0
	}

	maxAsync := float64(s.LecHours)
	if s.AsynchronousHours > maxAsync {
		return maxAsync
	}

	return s.AsynchronousHours
}

func (s Subject) EffectiveSynchronousLectureHours() float64 {
	sync := float64(s.LecHours) - s.EffectiveAsynchronousHours()
	if sync < 0 {
		return 0
	}
	return sync
}

func (s Subject) ComputedSynchronousHours() float64 {
	sync := s.TotalHours() - s.EffectiveAsynchronousHours()
	if sync < 0 {
		return 0
	}
	return sync
}

func (s *Subject) NormalizeAsyncConfig() {
	s.SubjectType = s.NormalizedSubjectType()
	s.AsynchronousHours = s.EffectiveAsynchronousHours()

	if s.IsLaboratoryType() {
		s.AsynchronousHours = 0
	}

	s.SynchronousHours = s.ComputedSynchronousHours()
}

func hoursToSlots(hours float64) int {
	if hours <= 0 {
		return 0
	}

	slots := int(math.Round(hours * float64(Const.N_HOUR_TIME_SLOTS)))
	if slots < 0 {
		return 0
	}

	return slots
}

// SlotsToAssign returns the number of 30-minute slots consumed in the weekly
// grid, excluding asynchronous hours.
func (s Subject) SlotsToAssign() int {
	syncHours := s.ComputedSynchronousHours()
	if syncHours <= 0 {
		syncHours = s.TotalHours()
	}
	return hoursToSlots(syncHours)
}

// SlotsToAssignByClassType returns grid slots for 0=lecture, 1=laboratory.
func (s Subject) SlotsToAssignByClassType(classType int) int {
	switch classType {
	case 0:
		return hoursToSlots(s.EffectiveSynchronousLectureHours())
	case 1:
		return hoursToSlots(float64(s.LabHours))
	default:
		return 0
	}
}

// NOTE: there are some subjects that has 9 class hours,
// so it is not possible to store the records in just 3 bits
