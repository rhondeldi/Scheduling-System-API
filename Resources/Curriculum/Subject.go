package Curriculum

const LEC_HOURS_BIT_MASK uint32 = 0b11111111
const LAB_HOURS_BIT_MASK uint32 = 0b11111111 << 8
const FLAG_BIT_MASK uint32 = 0b1111111111111111 << 16

// use to read and write subject information from and to the database.
type Subject struct {
	// this should never be zero, zero means empty, none or nothing.
	ID       uint16 `json:"ID,omitempty" bson:"ID,omitempty"`
	Code     string `json:"Code" bson:"Code"`
	Name     string `json:"Name" bson:"Name"`
	LecHours uint8  `json:"LecHours" bson:"LecHours"`
	LabHours uint8  `json:"LabHours" bson:"LabHours"`

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

// NOTE: there are some subjects that has 9 class hours,
// so it is not possible to store the records in just 3 bits
