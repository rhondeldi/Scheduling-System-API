package GeneticAlgorithm

import (
	"fmt"
	"log"
	"reflect"
	"slices"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
)

// struct type containing members that is use to track resource allocation/utilization in a schedule.
type EncodingResource struct {
	IsSchedIdxToSubIdToSkip map[uint16]map[uint16]bool
	DeptIdToInstructors     map[uint16][]Instructors.Instructor
	DeptIdToRoomtypeToRooms map[uint16]map[uint16][]Rooms.Room

	IdToInstructor map[uint16]*Instructors.Instructor // flatten `DeptIdToInstructors`
	IdToRoom       map[uint16]*Rooms.Room             // flatten `DeptIdToRoomtypeToRooms`
}

func (s *EncodingResource) MakeCopy() (*EncodingResource, error) {
	////////////////////////////////////////////////////////////////////////////////////////

	is_sched_idx_to_sub_id_to_skip := make(map[uint16]map[uint16]bool)

	for out_k, out_v := range s.IsSchedIdxToSubIdToSkip {
		is_sched_idx_to_sub_id_to_skip[out_k] = make(map[uint16]bool)

		for in_k, in_v := range out_v {
			is_sched_idx_to_sub_id_to_skip[out_k][in_k] = in_v
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////

	dept_id_to_room_type_to_rooms := make(map[uint16]map[uint16][]Rooms.Room)

	for out_k, out_v := range s.DeptIdToRoomtypeToRooms {
		dept_id_to_room_type_to_rooms[out_k] = make(map[uint16][]Rooms.Room)

		for in_k, in_v := range out_v {
			dept_id_to_room_type_to_rooms[out_k][in_k] = make([]Rooms.Room, len(in_v))
			copies := copy(dept_id_to_room_type_to_rooms[out_k][in_k], in_v)

			if copies != len(in_v) {
				log.Printf("copies : %d\tlen(dept_id_to_room_type_to_rooms[out_k][in_k] = %d/%d = in_v)\n", copies, len(dept_id_to_room_type_to_rooms[out_k][in_k]), len(in_v))
				return nil, fmt.Errorf("slice elements copied %d, internal department id to rooms map copy operation failed in generate new individual function", copies)
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////

	dept_id_to_instructors := make(map[uint16][]Instructors.Instructor)

	for k, v := range s.DeptIdToInstructors {
		dept_id_to_instructors[k] = make([]Instructors.Instructor, len(v))
		copies := copy(dept_id_to_instructors[k], v)
		if copies != len(v) {
			return nil, fmt.Errorf("slice elements copied %d, internal department id to instructors map copy operation failed in generate new individual function", copies)
		}
	}

	//////////////////////////////////////////////////////////////////////////////////////
	//                              FLATTEN ENCODING RESOURCES
	//////////////////////////////////////////////////////////////////////////////////////

	room_id_to_room := make(map[uint16]*Rooms.Room)

	for out_key, out_v := range dept_id_to_room_type_to_rooms {
		for in_key, in_v := range out_v {
			for room_idx, room := range in_v {
				room_id_to_room[room.RoomID] = &dept_id_to_room_type_to_rooms[out_key][in_key][room_idx]
			}
		}
	}

	instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

	for k, v := range dept_id_to_instructors {
		for instructor_idx, instructor := range v {
			instructor_id_to_instructor[instructor.InstructorID] = &dept_id_to_instructors[k][instructor_idx]
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////

	return &EncodingResource{
		IsSchedIdxToSubIdToSkip: is_sched_idx_to_sub_id_to_skip,
		DeptIdToInstructors:     dept_id_to_instructors,
		DeptIdToRoomtypeToRooms: dept_id_to_room_type_to_rooms,

		IdToInstructor: instructor_id_to_instructor,
		IdToRoom:       room_id_to_room,
	}, nil
}

func IsEqualEncodingResource(a, b *EncodingResource) bool {

	////////////////////////////////////////////////////////////////////////////////////////

	remove_false_subject_to_skip := func(m map[uint16]map[uint16]bool) map[uint16][]uint16 {
		out := make(map[uint16][]uint16, len(m))

		for usi, subject_id_to_skip := range m {

			for subject_id, to_skip := range subject_id_to_skip {
				if !to_skip {
					continue
				}

				out[usi] = append(out[usi], subject_id)
			}

			slices.Sort(out[usi])
		}

		return out
	}

	skip_subjects_a := remove_false_subject_to_skip(a.IsSchedIdxToSubIdToSkip)
	skip_subjects_b := remove_false_subject_to_skip(b.IsSchedIdxToSubIdToSkip)

	if !reflect.DeepEqual(skip_subjects_a, skip_subjects_b) {
		log.Print("IsEqualEncodingResource: not equal IsSchedIdxToSubIdToSkip")
		return false
	}

	////////////////////////////////////////////////////////////////////////////////////////

	flatten_instructors := func(m map[uint16][]Instructors.Instructor) []Instructors.Instructor {
		out := make([]Instructors.Instructor, 0)

		for _, instructors := range m {
			out = append(out, instructors...)
		}

		sort.Slice(out, func(i, j int) bool {
			return out[i].InstructorID < out[j].InstructorID
		})

		return out
	}

	instructors_a := flatten_instructors(a.DeptIdToInstructors)
	instructors_b := flatten_instructors(b.DeptIdToInstructors)

	if len(instructors_a) != len(instructors_b) {
		log.Printf("IsEqualEncodingResource: not equal instructor length a(%d) != b(%d)", len(instructors_a), len(instructors_b))
		return false
	}

	for i := range len(instructors_a) {
		if instructors_a[i].InstructorID != instructors_b[i].InstructorID {
			log.Printf("Mismatch in InstructorID at index %d: %d != %d", i, instructors_a[i].InstructorID, instructors_b[i].InstructorID)
			return false
		}

		if instructors_a[i].DepartmentID != instructors_b[i].DepartmentID {
			log.Printf("Mismatch in DepartmentID at index %d: %d != %d", i, instructors_a[i].DepartmentID, instructors_b[i].DepartmentID)
			return false
		}

		if instructors_a[i].FirstName != instructors_b[i].FirstName {
			log.Printf("Mismatch in FirstName at index %d: %s != %s", i, instructors_a[i].FirstName, instructors_b[i].FirstName)
			return false
		}

		if instructors_a[i].MiddleInitial != instructors_b[i].MiddleInitial {
			log.Printf("Mismatch in MiddleInitial at index %d: %s != %s", i, instructors_a[i].MiddleInitial, instructors_b[i].MiddleInitial)
			return false
		}

		if instructors_a[i].LastName != instructors_b[i].LastName {
			log.Printf("Mismatch in LastName at index %d: %s != %s", i, instructors_a[i].LastName, instructors_b[i].LastName)
			return false
		}

		if instructors_a[i].AssignedSubjects != instructors_b[i].AssignedSubjects {
			log.Printf("Mismatch in AssignedSubjects at index %d: %d != %d", i, instructors_a[i].AssignedSubjects, instructors_b[i].AssignedSubjects)
			return false
		}

		if instructors_a[i].TotalTeachingHours != instructors_b[i].TotalTeachingHours {
			log.Printf("Mismatch in TotalTeachingHours at index %d: %f != %f", i, instructors_a[i].TotalTeachingHours, instructors_b[i].TotalTeachingHours)
			return false
		}

		if len(instructors_a[i].Time) != len(instructors_b[i].Time) {
			log.Printf("Mismatch in Time length at index %d: %d != %d", i, len(instructors_a[i].Time), len(instructors_b[i].Time))
			return false
		}

		for limb_a_idx, limb := range instructors_a[i].Time {
			if instructors_b[i].Time[limb_a_idx] != limb {
				log.Printf("Mismatch in Time at index %d, limb %d: %v != %v", i, limb_a_idx, limb, instructors_b[i].Time[limb_a_idx])
				return false
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////

	flatten_rooms := func(m map[uint16]map[uint16][]Rooms.Room) []Rooms.Room {
		out := make([]Rooms.Room, 0)

		for _, room_type_to_rooms := range m {
			for _, rooms := range room_type_to_rooms {
				out = append(out, rooms...)
			}
		}

		sort.Slice(out, func(i, j int) bool {
			return out[i].RoomID < out[j].RoomID
		})

		return out
	}

	rooms_a := flatten_rooms(a.DeptIdToRoomtypeToRooms)
	rooms_b := flatten_rooms(b.DeptIdToRoomtypeToRooms)

	if len(rooms_a) != len(rooms_b) {
		log.Printf("IsEqualEncodingResource: not equal room count a(%d) != b(%d)", len(rooms_a), len(rooms_b))
		return false
	}

	for i := range rooms_a {

		if rooms_a[i].RoomID != rooms_b[i].RoomID {
			log.Printf(
				"IsEqualEncodingResource: room mismatch at index %d - RoomID differs: a(%d) != b(%d)",
				i, rooms_a[i].RoomID, rooms_b[i].RoomID,
			)
			return false
		}

		if rooms_a[i].DepartmentID != rooms_b[i].DepartmentID {
			log.Printf(
				"IsEqualEncodingResource: room mismatch at index %d - DepartmentID differs: a(%d) != b(%d)",
				i, rooms_a[i].DepartmentID, rooms_b[i].DepartmentID,
			)
			return false
		}

		if rooms_a[i].RoomType != rooms_b[i].RoomType {
			log.Printf(
				"IsEqualEncodingResource: room mismatch at index %d - RoomType differs: a(%d) != b(%d)",
				i, rooms_a[i].RoomType, rooms_b[i].RoomType,
			)
			return false
		}

		if rooms_a[i].Capacity != rooms_b[i].Capacity {
			log.Printf(
				"IsEqualEncodingResource: room mismatch at index %d - Capacity differs: a(%d) != b(%d)",
				i, rooms_a[i].Capacity, rooms_b[i].Capacity,
			)
			return false
		}

		if rooms_a[i].Name != rooms_b[i].Name {
			log.Printf(
				"IsEqualEncodingResource: room mismatch at index %d - Name differs: a(%s) != b(%s)",
				i, rooms_a[i].Name, rooms_b[i].Name,
			)
			return false
		}

		if !reflect.DeepEqual(rooms_a[i].SharingDepartments, rooms_b[i].SharingDepartments) {
			log.Printf(
				"IsEqualEncodingResource: room mismatch at index %d - SharingDepartments differs: a(%v) != b(%v)",
				i, rooms_a[i].SharingDepartments, rooms_b[i].SharingDepartments,
			)

			return false
		}

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := range Const.N_DAILY_TIME_SLOTS {
				if rooms_a[i].GetTimeSlotClassCount(day, time_slot) != rooms_b[i].GetTimeSlotClassCount(day, time_slot) {
					log.Printf(
						"RIDX: %d, room_id: %d | d(%d), t(%d) => a(%d), b(%d)\n\na room:\n%+v\n\nb room:\n%+v\n\n",
						i, rooms_a[i].RoomID,
						day, time_slot,
						rooms_a[i].GetTimeSlotClassCount(day, time_slot),
						rooms_b[i].GetTimeSlotClassCount(day, time_slot),
						rooms_a[i], rooms_b[i],
					)
					return false
				}
			}
		}
	}

	return true
}

/*
@param - `default_empty_encoding_resource` - should strictly

use to generate an `EncodingResource` for a `UniTimeTables`, can be used for user
configured university schedules or university schedules loaded from the persistence
since they don't have any associated `EncodingResource` instance.

NOTE TO SELF IN THE FUTURE:

if you want this function to support department specific encoding resource generation... DON'T DO IT!
because generating encoding resource can't be department specific, to give an example,
general instructors can be assigned to multiple class / section on different departments (like FITT teachers),
if you only generate encoding resource that is department specific, the other allocations to different
departments will not be reflected, this also applies to general rooms.
*/
func GenerateEncodingResourceFromUniTimeTable(
	university_schedules Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	selected_semester int,
	default_empty_encoding_resource *EncodingResource,
) (*EncodingResource, error) {

	encode_resource, err_make_copy := default_empty_encoding_resource.MakeCopy()

	if err_make_copy != nil {
		return nil, err_make_copy
	}

	//////////////////////////////////////////////////////////////////////////////////////
	//                           RE-CREATE ENCODING RESOURCE DATA
	//////////////////////////////////////////////////////////////////////////////////////

	// subject id -> credit units, used to seed each instructor's AssignedUnits from
	// the (frozen) base schedule so the generation-time unit cap accounts for an
	// instructor's load in OTHER departments / already-placed sections too.
	subject_id_to_units := make(map[uint16]uint8)
	for _, curriculum := range curriculums {
		for _, year_level := range curriculum.YearLevels {
			for _, semester := range year_level.Semesters {
				for _, subject := range semester.Subjects {
					subject_id_to_units[subject.ID] = subject.Units
				}
			}
		}
	}

	IterateSectionsWeekSchedule(university_schedules, curriculums, selected_semester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			// If there is no week schedule for this section, skip it.
			if values.WeekSched == nil {
				return IterProceed
			}

			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
					subject_id := values.WeekSched[day].GetTimeSlot(time_slot).GetSubjectID()

					if subject_id != 0 {
						instructor_id := values.WeekSched[day].GetTimeSlot(time_slot).GetInstructorID()
						room_id := values.WeekSched[day].GetTimeSlot(time_slot).GetRoomID()

						if instructor_id == 0 {
							log.Panic("there should be an instructor allocation here, why there is none?")
						}

						if room_id == 0 {
							log.Panic("there should be a room allocation here, why there is none?")
						}

						encode_resource.IdToInstructor[instructor_id].Time.SetAvailability(false, day, time_slot)
						encode_resource.IdToRoom[room_id].IncTimeSlotClassCount(day, time_slot)

						_, has_sched_idx := encode_resource.IsSchedIdxToSubIdToSkip[uint16(indicies.Usi)]

						if !has_sched_idx {
							encode_resource.IsSchedIdxToSubIdToSkip[uint16(indicies.Usi)] = make(map[uint16]bool)
						}

						_, has_subject_id := encode_resource.IsSchedIdxToSubIdToSkip[uint16(indicies.Usi)][subject_id]

						if !has_subject_id {
							encode_resource.IsSchedIdxToSubIdToSkip[uint16(indicies.Usi)][subject_id] = true
							encode_resource.IdToInstructor[instructor_id].AssignedSubjects++
							encode_resource.IdToInstructor[instructor_id].AssignedUnits += uint16(subject_id_to_units[subject_id])
						}

						encode_resource.IdToInstructor[instructor_id].TotalTeachingHours += (1.0 / Const.N_HOUR_TIME_SLOTS)
					}
				} // ------------- end of time_slot loop -------------
			} // ------------- end of day loop -------------

			return IterProceed
		},
	)

	return encode_resource, nil
}

// reads the default values of `EncodingResource` saved in a persistence instance.
func ReadDefaultEncodingResource(resource_persistence *StorageResources.Persistence) (*EncodingResource, error) {

	rooms, err_read_rooms := resource_persistence.ReaderService.ReadAllRooms()

	if err_read_rooms != nil {
		return nil, err_read_rooms
	}

	dept_id_to_room_type_to_rooms := GenerateMapDeptIdToRoomTypeToRooms(rooms)

	instructors, err_read_instructors := resource_persistence.ReaderService.ReadAllInstructors()

	if err_read_instructors != nil {
		return nil, err_read_instructors
	}

	dept_id_to_instructors := GenerateMapDeptIdToInstructors(instructors)

	//////////////////////////////////////////////////////////////////////////////////////
	//                              FLATTEN ENCODING RESOURCES
	//////////////////////////////////////////////////////////////////////////////////////

	room_id_to_room := make(map[uint16]*Rooms.Room)

	for out_key, out_v := range dept_id_to_room_type_to_rooms {
		for in_key, in_v := range out_v {
			for room_idx, room := range in_v {
				room_id_to_room[room.RoomID] = &dept_id_to_room_type_to_rooms[out_key][in_key][room_idx]
			}
		}
	}

	instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

	for k, v := range dept_id_to_instructors {
		for instructor_idx, instructor := range v {
			instructor_id_to_instructor[instructor.InstructorID] = &dept_id_to_instructors[k][instructor_idx]
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////

	return &EncodingResource{
		IsSchedIdxToSubIdToSkip: make(map[uint16]map[uint16]bool),
		DeptIdToInstructors:     dept_id_to_instructors,
		DeptIdToRoomtypeToRooms: dept_id_to_room_type_to_rooms,

		IdToInstructor: instructor_id_to_instructor,
		IdToRoom:       room_id_to_room,
	}, nil
}

const (
	sizeUint16       = 2 // bytes
	sizeBool         = 1 // bytes
	sizeInt          = 8 // bytes on 64-bit systems
	sizeFloat32      = 4
	sizeStringHeader = 16 // ptr + len
	sizeSliceHeader  = 24 // ptr + len + cap
)

func (e *EncodingResource) EstimateMemoryUsageInBytes() uint64 {

	instructor_size := func(inst Instructors.Instructor) uint64 {
		var total_instructor_byte_size uint64

		total_instructor_byte_size += sizeUint16                                             // InstructorID
		total_instructor_byte_size += sizeUint16                                             // DepartmentID
		total_instructor_byte_size += sizeInt                                                // AssignedSubjects
		total_instructor_byte_size += sizeFloat32                                            // TotalTeachingHours
		total_instructor_byte_size += uint64(Instructors.INSTRUCTOR_TIME_SLOT_MAP_LIMBS) * 8 // [3]uint64

		total_instructor_byte_size += uint64(len(inst.FirstName)) + sizeStringHeader
		total_instructor_byte_size += uint64(len(inst.MiddleInitial)) + sizeStringHeader
		total_instructor_byte_size += uint64(len(inst.LastName)) + sizeStringHeader

		return total_instructor_byte_size
	}

	room_size := func(r Rooms.Room) uint64 {
		var total_room_size uint64

		total_room_size += sizeUint16 * 4                             // RoomID, DeptID, Capacity, RoomType
		total_room_size += uint64(Rooms.TIME_SLOT_CLASS_COUNTER_SIZE) // [72]uint8

		total_room_size += sizeStringHeader + uint64(len(r.Name))

		total_room_size += sizeSliceHeader
		total_room_size += uint64(len(r.SharingDepartments)) * sizeUint16

		return total_room_size
	}

	total := uint64(0)

	for _, sub_id_to_skip := range e.IsSchedIdxToSubIdToSkip {
		total += sizeUint16
		for range sub_id_to_skip {
			total += sizeUint16 + sizeBool
		}
	}

	for _, instructor_slice := range e.DeptIdToInstructors {
		total += sizeUint16 + sizeSliceHeader
		for _, instructor := range instructor_slice {
			total += instructor_size(instructor)
		}
	}

	for _, room_type_to_rooms := range e.DeptIdToRoomtypeToRooms {
		total += sizeUint16
		for _, room_slice := range room_type_to_rooms {
			total += sizeUint16 + sizeSliceHeader
			for _, room := range room_slice {
				total += room_size(room)
			}
		}
	}

	// add key and pointer sizes for IdToInstructor and IdToRoom

	total += uint64(sizeUint16+8) * uint64(len(e.IdToInstructor))
	total += uint64(sizeUint16+8) * uint64(len(e.IdToRoom))

	log.Printf("Total number of instructors: %d", len(e.IdToInstructor))
	log.Printf("Total number of rooms: %d", len(e.IdToRoom))
	log.Printf("Total number of classes/sections: %d", len(e.IsSchedIdxToSubIdToSkip))
	log.Printf("Estimated memory usage of EncodingResource: %d bytes", total)

	return total
}
