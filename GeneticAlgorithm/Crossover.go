package GeneticAlgorithm

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

const CROSSOVER_DOMINANT_GENE int = 75 // % - percentage to be likely that the dominant parent's gene will be use during crossover

func Crossover(
	parent1, parent2 Schedule.UniTimeTables, default_encoding_resource *EncodingResource,
	curriculums []Curriculum.Curriculum, rooms []Rooms.Room, selected_semester int,
	dept_id_to_department map[uint16]Departments.Department,
	department_to_encode map[uint16]bool,
	instructor_id_to_instructor map[uint16]*Instructors.Instructor,
	resource_persistence *StorageResources.Persistence,
) (*SchedAndResources, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))

	if len(parent1) != len(parent2) {
		log.Printf(
			"Crossover [parent-length-error]: parents must have the same length, parent 1 length: %d, parent 2 length: %d",
			len(parent1), len(parent2),
		)

		return nil, fmt.Errorf(
			"crossover error, parents must have the same length, parent 1 length: %d, parent 2 length: %d",
			len(parent1), len(parent2),
		)
	}

	total_encoding_tries := 0
	successful_base_parent_encoded := 0
	successful_fallback_parent_encoded := 0
	failed_parents_encoding := 0

	offspring := make(Schedule.UniTimeTables, len(parent1))

	// copy other department subjects to the offspring, it doesn't matter if we use parent 1 or 2
	// both parents should have the same subjects allocated in other departments

	copied_week_time_table := copy(offspring, parent1)

	if copied_week_time_table != len(parent1) {
		log.Printf("Crossover [copy-error-uni-sched]: unable to copy other department subjects to the offspring")

		return nil, errors.New(
			"crossover error, unable to copy other department subjects to the offspring",
		)
	}

	if len(department_to_encode) != 1 {
		log.Printf("Crossover [copy-error-encoding-resource]: multiple department to encode is not supported yet by the crossover function")

		return nil, errors.New(
			"crossover error, multiple department to encode is not supported yet by the crossover function",
		)
	}

	// we only clear the current department from the offspring where the subjects would be different

	for department_id, is_to_encode := range department_to_encode {
		if is_to_encode {
			ClearDepartmentSchedule(offspring, curriculums, department_id, selected_semester)
		}
	}

	offspring_encode_resource, err_gen_encoding_resource := GenerateEncodingResourceFromUniTimeTable(
		offspring, curriculums, selected_semester, default_encoding_resource,
	)

	if err_gen_encoding_resource != nil {
		return nil, fmt.Errorf("crossover generate encoding error, caused by : %s", err_gen_encoding_resource.Error())
	}

	is_err_to_return := false
	var return_err error

	IterateSectionsWeekSchedule(nil, curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		is_to_encode, has_key := department_to_encode[values.Curriculum.DepartmentID]

		if !(is_to_encode && has_key) {
			return IterProceed
		}

		parent_1_subjects := parent1[indicies.Usi].GetWeekSubjectsJSON()
		parent_2_subjects := parent2[indicies.Usi].GetWeekSubjectsJSON()

		if len(parent_1_subjects) != len(parent_2_subjects) {
			is_err_to_return = true

			log.Printf(
				"Crossover [error-subjects-different-time-slot-block-counts] : parent 1 subjects: %d, parent 2 subjects: %d, possible cause by wrong university schedule indexing order",
				len(parent_1_subjects), len(parent_2_subjects),
			)

			return_err = fmt.Errorf(
				"error parent subjects have different subject time slot block counts, parent 1 subjects: %d, parent 2 subjects: %d, possible cause by wrong university schedule indexing order",
				len(parent_1_subjects), len(parent_2_subjects),
			)

			fmt.Print("\n\nParent 1:\n\n")

			Utils.PrettyPrint(parent_1_subjects)

			fmt.Print("\n\nParent 2:\n\n")

			Utils.PrettyPrint(parent_2_subjects)

			fmt.Print("\n\n")

			return IterBreakCurriculumLoop
		}

		sort.Slice(parent_1_subjects, func(i, j int) bool {
			if parent_1_subjects[i].SubjectID == parent_1_subjects[j].SubjectID {
				return parent_1_subjects[i].TimeSlotSize < parent_1_subjects[j].TimeSlotSize
			}

			return parent_1_subjects[i].SubjectID < parent_1_subjects[j].SubjectID
		})

		sort.Slice(parent_2_subjects, func(i, j int) bool {
			if parent_2_subjects[i].SubjectID == parent_2_subjects[j].SubjectID {
				return parent_2_subjects[i].TimeSlotSize < parent_2_subjects[j].TimeSlotSize
			}

			return parent_2_subjects[i].SubjectID < parent_2_subjects[j].SubjectID
		})

		// cheap sanity check - keep this

		for i := 0; i < max(len(parent_1_subjects), len(parent_2_subjects)); i++ {
			is_equal_subject_id := parent_1_subjects[i].SubjectID == parent_2_subjects[i].SubjectID
			is_equal_time_slot_size := parent_1_subjects[i].TimeSlotSize == parent_2_subjects[i].TimeSlotSize

			if !is_equal_subject_id {
				log.Print("Crossover: [unexpected-error-different-subjects]: parents have contain different subjects")

				is_err_to_return = true
				return_err = errors.New("crossover unexpected error, parents have contain different subjects")

				return IterBreakCurriculumLoop
			}

			if !is_equal_time_slot_size {
				log.Print("Crossover: [unexpected-error-not-equal-time-slots]: parent subjects have different time slot sizes")

				is_err_to_return = true
				return_err = errors.New("crossover unexpected error, parent subjects have different time slot sizes")

				return IterBreakCurriculumLoop
			}
		}

		for i := 0; i < len(parent_1_subjects); i++ {

			is_equal_subject_id := (parent_1_subjects[i].SubjectID == parent_2_subjects[i].SubjectID)
			is_equal_subject_time_slot_size := (parent_1_subjects[i].TimeSlotSize == parent_2_subjects[i].TimeSlotSize)

			if !is_equal_subject_id {

				log.Printf(
					"Crossover: [error-different-subjects] parents must have the same subject IDs, parent 1 subject ID: %d, parent 2 subject ID: %d",
					parent_1_subjects[i].SubjectID, parent_2_subjects[i].SubjectID,
				)

				is_err_to_return = true
				return_err = fmt.Errorf(
					"parents must have the same subject IDs, parent 1 subject ID: %d, parent 2 subject ID: %d",
					parent_1_subjects[i].SubjectID, parent_2_subjects[i].SubjectID,
				)

				return IterBreakCurriculumLoop
			}

			if !is_equal_subject_time_slot_size {

				log.Printf(
					"Crossover: [error-not-equal-time-slots] parents must have the same subject time slot size, parent 1 subject time slot size: %d, parent 2 subject time slot size: %d",
					parent_1_subjects[i].TimeSlotSize, parent_2_subjects[i].TimeSlotSize,
				)

				is_err_to_return = true
				return_err = fmt.Errorf(
					"parents must have the same subject time slot size, parent 1 subject time slot size: %d, parent 2 subject time slot size: %d",
					parent_1_subjects[i].TimeSlotSize, parent_2_subjects[i].TimeSlotSize,
				)

				return IterBreakCurriculumLoop
			}

			total_encoding_tries++

			//////////////////////////////////////////////////////////////////////////////////////////////
			//                             SELECT PARENT GENE TO INHERIT
			//////////////////////////////////////////////////////////////////////////////////////////////

			var base_parent_subjects []Schedule.TimeSlotSubjectJSON
			var fallback_parent_subjects []Schedule.TimeSlotSubjectJSON

			var dominant_parent_subjects []Schedule.TimeSlotSubjectJSON
			var recessive_parent_subjects []Schedule.TimeSlotSubjectJSON

			parent1_week_sched_fitness := MeasureWeekTimeTableBasicFitness(parent1[indicies.Usi])
			parent2_week_sched_fitness := MeasureWeekTimeTableBasicFitness(parent2[indicies.Usi])

			if parent1_week_sched_fitness > parent2_week_sched_fitness {
				dominant_parent_subjects = parent_1_subjects
				recessive_parent_subjects = parent_2_subjects
			} else {
				dominant_parent_subjects = parent_2_subjects
				recessive_parent_subjects = parent_1_subjects
			}

			if rng.Int31n(100) <= int32(CROSSOVER_DOMINANT_GENE) {
				base_parent_subjects = dominant_parent_subjects
				fallback_parent_subjects = recessive_parent_subjects
			} else {
				base_parent_subjects = recessive_parent_subjects
				fallback_parent_subjects = dominant_parent_subjects
			}

			//////////////////////////////////////////////////////////////////////////////////////////////
			//                             ENCODE THE BASE PARENT SUBJECTS
			//////////////////////////////////////////////////////////////////////////////////////////////

			base_parent_result := inherit_trait_from_a_parent(
				i, indicies.Usi,
				offspring, offspring_encode_resource,
				base_parent_subjects,
			)

			if base_parent_result.success {
				successful_base_parent_encoded++

				if base_parent_result.has_extended_subject {
					i++
				}

				continue // to next subject
			}

			//////////////////////////////////////////////////////////////////////////////////////////////
			//                            ENCODE THE FALLBACK PARENT SUBJECTS
			//////////////////////////////////////////////////////////////////////////////////////////////

			fallback_parent_result := inherit_trait_from_a_parent(
				i, indicies.Usi,
				offspring, offspring_encode_resource,
				fallback_parent_subjects,
			)

			if fallback_parent_result.has_extended_subject {
				i++
			}

			if fallback_parent_result.success {
				successful_fallback_parent_encoded++
				continue // to next subject
			}

			// if both parents failed to encode, we will just try to re-encode to repair it later
			failed_parents_encoding++
		}

		return IterProceed
	})

	if os.Getenv("LOG_MODE") == "verbose" {
		log.Printf(
			"Crossover: total encoding tries: %d, successful base parent encoded: %d, successful fallback parent encoded: %d, failed parents encoding: %d",
			total_encoding_tries, successful_base_parent_encoded, successful_fallback_parent_encoded, failed_parents_encoding,
		)
	}

	if is_err_to_return {
		return nil, fmt.Errorf("crossover error, caused by : %s", return_err.Error())
	}

	if failed_parents_encoding > 0 {
		// re-encode the schedule - fillup missing time slots

		repaired_sched, repaired_encoding_resource, err_repair_encoding := EncodeIndividualGenome(
			offspring, curriculums,
			dept_id_to_department, offspring_encode_resource,
			department_to_encode, selected_semester, 0,
		)

		if err_repair_encoding != nil {
			return nil, fmt.Errorf("crossover completion error, caused by : %s", err_repair_encoding.Error())
		}

		return &SchedAndResources{
			UniSched:  repaired_sched,
			Resources: repaired_encoding_resource,
		}, nil
	}

	return &SchedAndResources{
		UniSched:  offspring,
		Resources: offspring_encode_resource,
	}, nil
}

type inherit_trait_result struct {
	success              bool
	has_extended_subject bool
}

func inherit_trait_from_a_parent(
	i, usi int,
	offspring Schedule.UniTimeTables, offspring_encode_resource *EncodingResource,
	json_subjects []Schedule.TimeSlotSubjectJSON,
) inherit_trait_result {
	subject := &json_subjects[i]

	id_to_room := offspring_encode_resource.IdToRoom
	id_to_instructor := offspring_encode_resource.IdToInstructor

	has_extended_subject := false

	if i+1 < len(json_subjects) {
		if (json_subjects[i+1].SubjectID == subject.SubjectID) && (json_subjects[i+1].InstructorID == subject.InstructorID) {
			has_extended_subject = true
		}
	}

	is_first_target_time_slot_free := true
	is_second_target_time_slot_free := true

	for j := 0; j < subject.TimeSlotSize; j++ {
		is_time_slot_available := offspring[usi][subject.Day][subject.StartingTimeSlot+j].GetSubjectID() == 0
		is_instructor_available := id_to_instructor[subject.InstructorID].Time.GetAvailability(subject.Day, subject.StartingTimeSlot+j)
		is_room_available := id_to_room[subject.RoomID].GetTimeSlotClassCount(subject.Day, subject.StartingTimeSlot+j) < uint8(id_to_room[subject.RoomID].Capacity)

		if !(is_time_slot_available && is_instructor_available && is_room_available) {
			is_first_target_time_slot_free = false
			break
		}
	}

	if has_extended_subject {
		subj_extend := &json_subjects[i+1]

		for j := 0; j < subj_extend.TimeSlotSize; j++ {
			is_time_slot_available := offspring[usi][subj_extend.Day][subj_extend.StartingTimeSlot+j].GetSubjectID() == 0
			is_instructor_available := id_to_instructor[subj_extend.InstructorID].Time.GetAvailability(subj_extend.Day, subj_extend.StartingTimeSlot+j)
			is_room_available := id_to_room[subj_extend.RoomID].GetTimeSlotClassCount(subj_extend.Day, subj_extend.StartingTimeSlot+j) < uint8(id_to_room[subj_extend.RoomID].Capacity)

			if !(is_time_slot_available && is_instructor_available && is_room_available) {
				is_second_target_time_slot_free = false
				break
			}
		}

		if is_first_target_time_slot_free && is_second_target_time_slot_free {
			for j := 0; j < subject.TimeSlotSize; j++ {
				offspring[usi][subject.Day][subject.StartingTimeSlot+j].SetSubjectID(subject.SubjectID)
				offspring[usi][subject.Day][subject.StartingTimeSlot+j].SetInstructorID(subject.InstructorID)
				offspring[usi][subject.Day][subject.StartingTimeSlot+j].SetRoomID(subject.RoomID)

				id_to_instructor[subject.InstructorID].Time.SetAvailability(false, subject.Day, subject.StartingTimeSlot+j)
				id_to_room[subject.RoomID].IncTimeSlotClassCount(subject.Day, subject.StartingTimeSlot+j)
			}

			for j := 0; j < subj_extend.TimeSlotSize; j++ {
				offspring[usi][subj_extend.Day][subj_extend.StartingTimeSlot+j].SetSubjectID(subj_extend.SubjectID)
				offspring[usi][subj_extend.Day][subj_extend.StartingTimeSlot+j].SetInstructorID(subj_extend.InstructorID)
				offspring[usi][subj_extend.Day][subj_extend.StartingTimeSlot+j].SetRoomID(subj_extend.RoomID)

				id_to_instructor[subj_extend.InstructorID].Time.SetAvailability(false, subj_extend.Day, subj_extend.StartingTimeSlot+j)
				id_to_room[subj_extend.RoomID].IncTimeSlotClassCount(subj_extend.Day, subj_extend.StartingTimeSlot+j)

			}

			id_to_instructor[subj_extend.InstructorID].AssignedSubjects++
			id_to_instructor[subj_extend.InstructorID].TotalTeachingHours += float32(subject.TimeSlotSize+subj_extend.TimeSlotSize) / float32(Const.N_HOUR_TIME_SLOTS)

			if _, has_sched_idx := offspring_encode_resource.IsSchedIdxToSubIdToSkip[uint16(usi)]; !has_sched_idx {
				offspring_encode_resource.IsSchedIdxToSubIdToSkip[uint16(usi)] = make(map[uint16]bool)
			}

			offspring_encode_resource.IsSchedIdxToSubIdToSkip[uint16(usi)][subject.SubjectID] = true

			return inherit_trait_result{
				success:              true,
				has_extended_subject: true,
			}
		}
	} else if is_first_target_time_slot_free {
		for j := 0; j < subject.TimeSlotSize; j++ {
			offspring[usi][subject.Day][subject.StartingTimeSlot+j].SetSubjectID(subject.SubjectID)
			offspring[usi][subject.Day][subject.StartingTimeSlot+j].SetInstructorID(subject.InstructorID)
			offspring[usi][subject.Day][subject.StartingTimeSlot+j].SetRoomID(subject.RoomID)

			id_to_instructor[subject.InstructorID].Time.SetAvailability(false, subject.Day, subject.StartingTimeSlot+j)
			id_to_room[subject.RoomID].IncTimeSlotClassCount(subject.Day, subject.StartingTimeSlot+j)
		}

		if _, has_sched_idx := offspring_encode_resource.IsSchedIdxToSubIdToSkip[uint16(usi)]; !has_sched_idx {
			offspring_encode_resource.IsSchedIdxToSubIdToSkip[uint16(usi)] = make(map[uint16]bool)
		}

		id_to_instructor[subject.InstructorID].AssignedSubjects++
		id_to_instructor[subject.InstructorID].TotalTeachingHours += float32(subject.TimeSlotSize) / float32(Const.N_HOUR_TIME_SLOTS)

		offspring_encode_resource.IsSchedIdxToSubIdToSkip[uint16(usi)][subject.SubjectID] = true

		return inherit_trait_result{
			success:              true,
			has_extended_subject: false,
		}
	}

	return inherit_trait_result{
		success:              false,
		has_extended_subject: has_extended_subject,
	}
}
