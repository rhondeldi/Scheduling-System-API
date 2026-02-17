package Schedule

/*
Represents a scheduling unit containing a subject, instructor, and room.

* Each attribute is encoded as a 16-bit unsigned integer

# NOTE TO SELF:

you don't need to add another property here to determine
if a subject is a lecture or a laboratory subject, instead you can extract
that information from the room assigned to the time slot.
*/
type TimeSlot struct {
	// this should never be zero, zero means empty, none or nothing.
	subjectID uint16

	// this should never be zero, zero means empty, none or nothing.
	instructorID uint16

	// this should never be zero, zero means empty, none or nothing.
	roomID uint16
}

// ============================= CONSTRUCTOR =============================

// set methods returns error because they will be used and exposed in the api.

// set all values in the timeslot
func (time_slot *TimeSlot) Set(subject_id, instructor_id, room_id uint16) error {
	if err := time_slot.SetSubjectID(subject_id); err != nil {
		return err
	}

	if err := time_slot.SetInstructorID(instructor_id); err != nil {
		return err
	}

	if err := time_slot.SetRoomID(room_id); err != nil {
		return err
	}

	return nil
}

// ============================= GET VALUE METHODS =============================

// Retrieves the subject ID value of the TimeSlot, excluding the constraint flag.
func (time_slot *TimeSlot) GetSubjectID() uint16 {
	return time_slot.subjectID
}

// Retrieves the instructor ID value of the TimeSlot, excluding the constraint flag.
func (time_slot *TimeSlot) GetInstructorID() uint16 {
	return time_slot.instructorID
}

// Retrieves the room ID value of the TimeSlot, excluding the constraint flag.
func (time_slot *TimeSlot) GetRoomID() uint16 {
	return time_slot.roomID
}

// ============================= SET VALUE METHODS =============================

// Updates the subject ID value of the TimeSlot.
// If the value exceeds ATTRIBUTE_MAX_VALUE, an error is returned.
func (time_slot *TimeSlot) SetSubjectID(subject_id uint16) error {
	time_slot.subjectID = subject_id
	return nil
}

// Updates the instructor ID value of the TimeSlot.
// If the value exceeds ATTRIBUTE_MAX_VALUE, an error is returned.
func (time_slot *TimeSlot) SetInstructorID(instructor_id uint16) error {
	time_slot.instructorID = instructor_id
	return nil
}

// Updates the room ID value of the TimeSlot.
// If the value exceeds ATTRIBUTE_MAX_VALUE, an error is returned.
func (time_slot *TimeSlot) SetRoomID(room_id uint16) error {
	time_slot.roomID = room_id
	return nil
}
