package Rooms

import (
	"fmt"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

// the maximum room capacity.
const MAX_ROOM_CAPACITY int = 15 // = 0b1111 (4 bits only)

// we only need 4 bits to store up to 15 hours
const TIME_SLOT_CLASS_COUNTER_SIZE int = (Const.N_DAILY_TIME_SLOTS * Const.N_WEEKLY_SCHOOL_DAYS) / 2

type Room struct {
	RoomID       uint16 `json:"RoomID" bson:"RoomID"`
	DepartmentID uint16 `json:"DepartmentID" bson:"DepartmentID"` // a room that have a 0 department ID are rooms that are for every departments
	Capacity     uint16 `json:"Capacity" bson:"Capacity"`         // maximum numbers of classes or sections a room can hold in a single time slot (max value = MAX_ROOM_CAPACITY).
	RoomType     uint16 `json:"RoomType" bson:"RoomType"`         // determines the room type: 0 => lec, 1 => lab, 2 => gym.
	Name         string `json:"Name" bson:"Name"`

	SharingDepartments []uint16 `json:"SharingDepartments,omitempty" bson:"SharingDepartments,omitempty"`

	timeSlotClassCount [TIME_SLOT_CLASS_COUNTER_SIZE]uint8 // records the numbers of classes or sections allocated in the room for a specific timeslot
}

const ROOM_TYPE_LEC uint16 = 0
const ROOM_TYPE_LAB uint16 = 1
const ROOM_TYPE_GYM uint16 = 2

// ROOM_ID_ASYNC is a virtual room used for asynchronous classes.
const ROOM_ID_ASYNC uint16 = 0xFFFF
const ROOM_NAME_ASYNC = "ASYNC"

var ROOM_TYPE_NAMES [3]string = [3]string{"Lec", "Lab", "Gym"}

func IsAsyncRoomID(roomID uint16) bool {
	return roomID == ROOM_ID_ASYNC
}

// set the current number of classes or sections allocated in the room for a specific time slot.
func (room *Room) SetTimeSlotClassCount(day, time_slot int, class_count uint8) {
	idx_2D_to_1D := (day * Const.N_DAILY_TIME_SLOTS) + time_slot

	// in one byte or uint8 we can use the two set of 4-bits (higher and lower) to store two class or
	// section count for a specific time slot, hence we divide by the number of bits of uint8 to 2.
	limb_idx := idx_2D_to_1D / 2
	shift_multiplier := idx_2D_to_1D % 2

	room.timeSlotClassCount[limb_idx] &= (0b11110000 >> (4 * shift_multiplier))
	room.timeSlotClassCount[limb_idx] |= class_count << (4 * shift_multiplier)
}

// increase by 1 the current number of classes or sections allocated in the room for a specific time slot.
//
// warning incrementing until the max room capacity 15 would overflow the whole
// uint8 which cause serialized data corruption due to overflow of the uint8 type.
func (room *Room) IncTimeSlotClassCount(day, time_slot int) {
	idx_2D_to_1D := (day * Const.N_DAILY_TIME_SLOTS) + time_slot
	limb_idx := idx_2D_to_1D / 2
	shift_multiplier := idx_2D_to_1D % 2

	current_time_slot_allocation := int(room.GetTimeSlotClassCount(day, time_slot))

	if current_time_slot_allocation >= int(room.Capacity) {
		panic(fmt.Sprintf(
			"(%s - [id:%d, type:%d])IncTimeSlotClassCount(%d, %d) - cannot increment to larger room capacity : (capacity current/max = %d/%d)",
			room.Name, room.RoomID, room.RoomType, day, time_slot, room.GetTimeSlotClassCount(day, time_slot), room.Capacity,
		))
	}

	if current_time_slot_allocation >= MAX_ROOM_CAPACITY {
		panic(fmt.Sprintf(
			"(%s - [id:%d, type:%d])IncTimeSlotClassCount(%d, %d) - uint4 overflow, cannot increment to higher room capacity : (capacity current/max = %d/%d)",
			room.Name, room.RoomID, room.RoomType, day, time_slot, room.GetTimeSlotClassCount(day, time_slot), room.Capacity,
		))
	}

	room.timeSlotClassCount[limb_idx] += (0b1 << (4 * shift_multiplier))
}

// decrease by 1 the current number of classes or sections allocated in the room for a specific time slot.
//
// note: decrementing 0 values will not do anything.
func (room *Room) DecTimeSlotClassCount(day, time_slot int) {
	idx_2D_to_1D := (day * Const.N_DAILY_TIME_SLOTS) + time_slot
	limb_idx := idx_2D_to_1D / 2
	shift_multiplier := idx_2D_to_1D % 2

	current_time_slot_allocation := int(room.GetTimeSlotClassCount(day, time_slot))

	if current_time_slot_allocation == 0 {
		return
	}

	room.timeSlotClassCount[limb_idx] -= (0b1 << (4 * shift_multiplier))
}

// get the current number of classes or sections allocated in the room for a specific time slot.
func (room *Room) GetTimeSlotClassCount(day, time_slot int) uint8 {
	idx_2D_to_1D := (day * Const.N_DAILY_TIME_SLOTS) + time_slot
	limb_idx := idx_2D_to_1D / 2
	shift_multiplier := idx_2D_to_1D % 2

	return (room.timeSlotClassCount[limb_idx] & (0b1111 << (4 * shift_multiplier))) >> (4 * shift_multiplier)
}
