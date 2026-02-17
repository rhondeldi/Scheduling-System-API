package Instructors

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

// ====================================================================================
// convert an unsigned integer into 1 if it has any bit that is set to 1,
// or into 0 if all bits are set to 0, using only bitwise operations
// to compute value during compile time.

const BITSET_LIMB_WIDENESS = 64
const BITS_PER_BYTE = 8
const u8BitReduce uint8 = Const.N_WEEKLY_TIME_SLOTS % BITSET_LIMB_WIDENESS

// since modding by 64 will always produce results less than 64 which
// can fit in 8 bit wide unsigned integer, we can then right away start
// reducing at the first 8 bits instead from from all 64 bits.

const u4BitReduce = (u8BitReduce >> 4) | (u8BitReduce & uint8(0b1111))
const u2BitReduce = (u4BitReduce >> 2) | (u4BitReduce & uint8(0b11))
const u1BitReduce = (u2BitReduce >> 1) | (u2BitReduce & uint8(0b1))

// ====================================================================================

const INSTRUCTOR_TIME_SLOT_MAP_LIMBS = (Const.N_WEEKLY_TIME_SLOTS / 64) + u1BitReduce

// This type helps us manage the weekly availability of an instructor,
// allowing us to keep track which time slots are available or unavailable for them.
type InstructorTimeSlotBitMap [INSTRUCTOR_TIME_SLOT_MAP_LIMBS]uint64

// setting a bit to 1 means the instructor is available,
// and 0 if not for that corresponding bit time slot.
func (bitset *InstructorTimeSlotBitMap) SetAvailability(available bool, day, time_slot int) {
	if day < 0 || day >= Const.N_WEEKLY_SCHOOL_DAYS {
		panic(fmt.Sprintf(
			"SetAvailability(..., day int = %d,...) : invalid argument `day`, accepted values are only 0 to %d",
			day, Const.N_WEEKLY_SCHOOL_DAYS-1,
		))
	}

	if time_slot < 0 || time_slot >= Const.N_DAILY_TIME_SLOTS {
		panic(fmt.Sprintf(
			"SetAvailability(..., time_slot int = %d) : invalid argument `time_slot`, accepted values are only 0 to %d",
			time_slot, Const.N_DAILY_TIME_SLOTS-1,
		))
	}

	idx_2D_to_1D := (day * Const.N_DAILY_TIME_SLOTS) + time_slot
	limb_idx := idx_2D_to_1D / BITSET_LIMB_WIDENESS
	limb_bit_idx := idx_2D_to_1D % BITSET_LIMB_WIDENESS

	if available {
		bitset[limb_idx] &= ^(uint64(1) << limb_bit_idx)
	} else {
		bitset[limb_idx] |= (uint64(1) << limb_bit_idx)
	}
}

func (bitset *InstructorTimeSlotBitMap) GetAvailability(day, time_slot int) bool {
	if day < 0 || day >= Const.N_WEEKLY_SCHOOL_DAYS {
		panic(fmt.Sprintf(
			"GetAvailability(..., day int = %d,...) : invalid argument `day`, accepted values are only 0 to %d",
			day, Const.N_WEEKLY_SCHOOL_DAYS-1,
		))
	}

	if time_slot < 0 || time_slot >= Const.N_DAILY_TIME_SLOTS {
		panic(fmt.Sprintf(
			"GetAvailability(..., time_slot int = %d) : invalid argument `time_slot`, accepted values are only 0 to %d",
			time_slot, Const.N_DAILY_TIME_SLOTS-1,
		))
	}

	idx_2D_to_1D := (day * Const.N_DAILY_TIME_SLOTS) + time_slot
	limb_idx := idx_2D_to_1D / BITSET_LIMB_WIDENESS
	limb_bit_idx := idx_2D_to_1D % BITSET_LIMB_WIDENESS

	return ((bitset[limb_idx] >> limb_bit_idx) & uint64(1)) == 0
}

func (bitset *InstructorTimeSlotBitMap) Serialize() []byte {
	serialized := make([]byte, (INSTRUCTOR_TIME_SLOT_MAP_LIMBS * (BITSET_LIMB_WIDENESS / BITS_PER_BYTE)))

	for i := 0; i < int(INSTRUCTOR_TIME_SLOT_MAP_LIMBS); i++ {
		serialized_start_idx := (i * (BITSET_LIMB_WIDENESS / BITS_PER_BYTE))
		serialized_end_idx := serialized_start_idx + (BITSET_LIMB_WIDENESS / BITS_PER_BYTE)

		binary.LittleEndian.PutUint64(serialized[serialized_start_idx:serialized_end_idx], bitset[i])
	}

	return serialized
}

func (bitset *InstructorTimeSlotBitMap) Deserialize(serialized []byte) {
	for i := 0; i < int(INSTRUCTOR_TIME_SLOT_MAP_LIMBS); i++ {
		serialized_start_idx := (i * (BITSET_LIMB_WIDENESS / BITS_PER_BYTE))
		serialized_end_idx := serialized_start_idx + (BITSET_LIMB_WIDENESS / BITS_PER_BYTE)

		bitset[i] = binary.LittleEndian.Uint64(serialized[serialized_start_idx:serialized_end_idx])
	}
}

// convert the time InstructorTimeSlotBitMap integer array to string number  array.
func (bitset *InstructorTimeSlotBitMap) Stringify() []string {
	time_stringify := make([]string, 0)

	for _, limb := range bitset {
		time_stringify = append(time_stringify, strconv.FormatUint(limb, 10))
	}

	return time_stringify
}

func (bitset *InstructorTimeSlotBitMap) StringParse(string_slice []string) error {
	if len(string_slice) != len(bitset) {
		return errors.New("string array length mismatch for StringParse")
	}

	for i := range bitset {
		number, err_parse := strconv.ParseUint(string_slice[i], 10, 64)

		if err_parse != nil {
			return err_parse
		}

		bitset[i] = number
	}

	return nil
}
