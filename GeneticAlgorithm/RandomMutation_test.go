package GeneticAlgorithm_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageSchedule"
)

func TestRandomMutations(t *testing.T) {
	persistence := StorageResources.Persistence{
		ReaderService: &StorageResources.JsonReader{},
		WriterService: &StorageResources.JsonWriter{},
	}

	target_semester := GeneticAlgorithm.TERM_1ST_SEMESTER

	t.Logf("Semester : %d\n\n", target_semester)

	total_test_iterations := 512
	allowed_failure_rate := 0.8 // 80% error rate allowed.

	err_list_generation := make([]error, 0, 8)
	err_list_validation := make([]error, 0, 8)

	////////////////////////////////////////////////////////////////////////////////////////

	departments, err_all_departments := persistence.ReaderService.ReadAllDepartments()

	if err_all_departments != nil {
		t.Fatal(err_all_departments)
	}

	if target_semester == 0 {
		err_add_room := persistence.WriterService.CreateRoom(Rooms.Room{
			DepartmentID:       0,
			Capacity:           1,
			RoomType:           Rooms.ROOM_TYPE_LEC,
			Name:               "SHARED_BY_TED_AND_DOM",
			SharingDepartments: []uint16{3, 4},
		})

		if err_add_room != nil {
			t.Fatal("error adding room : ", err_add_room.Error())
		}
	}

	rooms, err_all_rooms := persistence.ReaderService.ReadAllRooms()

	if err_all_rooms != nil {
		t.Fatal(err_all_rooms)
	}

	var added_room *Rooms.Room

	if target_semester == 0 {
		for _, room := range rooms {
			if room.Name == "SHARED_BY_TED_AND_DOM" {
				added_room = &room
				break
			}
		}
	}

	curriculums, err_all_curriculums := persistence.ReaderService.ReadAllCurriculum()

	if err_all_curriculums != nil {
		t.Fatal(err_all_curriculums)
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	////////////////////////////////////////////////////////////////////////////////////////

	default_encoding_resource, err_read_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(&persistence)

	if err_read_default_encoding_resource != nil {
		t.Fatal(err_read_default_encoding_resource)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	for i := 0; i < total_test_iterations; i++ {
		if (i == 0) || (((i + 1) % 32) == 0) {
			fmt.Printf("Generating schedules (%d)..................................\n", (i + 1))
		}

		total_sections_calculated := Curriculum.GetTotalNumberOfSections(curriculums, target_semester)
		empty_university_schedule := GeneticAlgorithm.NewEmptyIndividual(curriculums, target_semester)

		if total_sections_calculated != len(empty_university_schedule) {
			t.Fatalf(
				"the calculated total sections for the semester index %d is %d, but the generated empty schedules only contains %d which is a mismatch",
				target_semester, total_sections_calculated, len(empty_university_schedule),
			)
		}

		university_schedules, encoding_resource, err := GeneticAlgorithm.EncodeIndividualGenome(
			empty_university_schedule,
			curriculums,
			dept_id_to_department, default_encoding_resource, nil,
			target_semester, 0,
		)

		if len(university_schedules) == 0 {
			t.Fatal("No university schedules generated")
		}

		if len(university_schedules) != len(empty_university_schedule) {
			t.Fatalf(
				"the generated total sections for the semester index %d is %d, but the generated empty schedules only contains %d which is a mismatch",
				target_semester, len(university_schedules), len(empty_university_schedule),
			)
		}

		if err != nil {
			t.Log(err)
			err_list_generation = append(err_list_generation, err)
		}

		if university_schedules == nil {
			t.Fatalf("returned a nil university schedule : loop iteration %d\n", i)
		}

		if university_schedules.IsEmpty() {
			t.Fatalf("returned an empty university schedule : loop iteration %d\n", i)
		}

		// t.Logf("Schedules Generated : %d,    Calculated Total Number Of Sections : %d", len(university_schedules), total_sections_calculated)

		err_vertical_validations := university_schedules.VerticalValidation(rooms)

		for _, e := range err_vertical_validations {
			t.Error(e)
			err_list_validation = append(err_list_validation, e)
		}

		///////////////////////

		generated_encoding_resource, err_gen_encode_resource := GeneticAlgorithm.GenerateEncodingResourceFromUniTimeTable(
			university_schedules, curriculums, target_semester, default_encoding_resource,
		)

		if err_gen_encode_resource != nil {
			t.Fatal(err_gen_encode_resource)
		}

		if encoding_resource != nil {
			if !GeneticAlgorithm.IsEqualEncodingResource(generated_encoding_resource, encoding_resource) {
				schedules_persistence := StorageSchedule.Persistence{SaveService: &StorageSchedule.JsonWriter{}}
				schedules_persistence.SaveService.SaveSchedules(university_schedules, target_semester)
				t.Fatal("generated encoding resource from bare university schedule is not equal to the produced encoding resource of GA")
			}
		}

		/////////////////

		if err == nil {
			err_horizontal_validations := GeneticAlgorithm.HorizontalValidation(university_schedules, curriculums, nil, target_semester)

			for _, e := range err_horizontal_validations {
				t.Fatal(e)
			}
		}

		///////////////// test : there should be no changes to the university schedules and encodingh resources when encoded again

		if encoding_resource != nil {
			re_university_schedules, re_encoding_resource, re_err := GeneticAlgorithm.EncodeIndividualGenome(
				university_schedules,
				curriculums,
				dept_id_to_department, encoding_resource, nil,
				target_semester, 0,
			)

			if len(university_schedules) == 0 {
				t.Fatal("Test No Changes : error : No university schedules generated")
			}

			if re_err != nil {
				t.Fatal("Test No Changes : error :", re_err)
			}

			if !reflect.DeepEqual(university_schedules, re_university_schedules) {
				t.Fatal("Test No Changes : error university schedules equal test failed")
			}

			if !GeneticAlgorithm.IsEqualEncodingResource(encoding_resource, re_encoding_resource) {
				t.Fatal("Test No Changes : error encoding resource equal test failed")
			}

			if err := GeneticAlgorithm.ValidateEncodingResource(university_schedules, encoding_resource, curriculums, target_semester); err != nil {
				t.Fatal(err)
			}

			for _, department := range departments {
				dept_to_encode := make(map[uint16]bool)
				dept_to_encode[department.DepartmentID] = true

				GeneticAlgorithm.ApplyRandomDaySwapTimeSlots(university_schedules, encoding_resource, curriculums, department.DepartmentID, target_semester)
				GeneticAlgorithm.ApplyRandomSubjectDaySwap(university_schedules, encoding_resource, curriculums, department.DepartmentID, target_semester)
				GeneticAlgorithm.ApplyRandomSubjectTimeSlotNudge(university_schedules, encoding_resource, curriculums, department.DepartmentID, target_semester)
				GeneticAlgorithm.ApplyRandomSubjectTimeSlotAndDayNudge(university_schedules, encoding_resource, curriculums, department.DepartmentID, target_semester)
			}

			if err_vv := university_schedules.VerticalValidation(rooms); len(err_vv) > 0 {
				for _, e := range err_vv {
					t.Fatal("after random mutation vertical validation error : ", e)
				}
			}

			if err_enc_resource_v := GeneticAlgorithm.ValidateEncodingResource(university_schedules, encoding_resource, curriculums, target_semester); err_enc_resource_v != nil {
				t.Fatal("after random mutation encoding resource error : ", err_enc_resource_v)
			}

			if err_hv := GeneticAlgorithm.HorizontalValidation(university_schedules, curriculums, nil, target_semester); len(err_hv) > 0 {
				for _, e := range err_hv {
					t.Fatal("after random mutation horizontal validation error : ", e)
				}
			}

			t.Log(".")

		}

		if target_semester == 0 {
			// test room shared by departments correctness

			if added_room == nil {
				t.Fatal("there is no added room found")
			}

			if added_room.RoomID > 60000 {
				t.Fatal("something is wrong with the added room, id value too high")
			}

			GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedules, curriculums, target_semester, nil, nil,
				func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {
					for day := range Const.N_WEEKLY_SCHOOL_DAYS {
						for time_slot := range Const.N_DAILY_TIME_SLOTS {
							has_found_added_room := values.WeekSched[day][time_slot].GetRoomID() == added_room.RoomID
							is_shared_to_department := slices.Contains(added_room.SharingDepartments, values.Curriculum.DepartmentID)

							if has_found_added_room && !is_shared_to_department {
								t.Fatalf("the added room that was supposed to be used only by TED and DOM is found in %s", dept_id_to_department[values.Curriculum.DepartmentID].Name)
							}
						}
					}

					return GeneticAlgorithm.IterProceed
				},
			)
		}
	}

	failed_individuals := float64(len(err_list_generation))
	successful_individuals := total_test_iterations - int(failed_individuals)

	if failed_individuals > (float64(total_test_iterations) * allowed_failure_rate) {
		t.Errorf(
			"total of %d fails (%.2f%%) and %d success (%.2f%%) out of the %d populations generated which is above the maximum error treashold of %.2f%%",
			int(failed_individuals), failed_individuals/float64(total_test_iterations)*100.0,
			successful_individuals, float64(successful_individuals)/float64(total_test_iterations)*100.0,
			total_test_iterations, allowed_failure_rate*100.0,
		)
	} else {
		fmt.Printf(
			"total of %d fails (%.2f%%) and %d success (%.2f%%) out of the %d populations generated which is below the maximum error treashold of %.2f%%\n",
			int(failed_individuals), failed_individuals/float64(total_test_iterations)*100.0,
			successful_individuals, float64(successful_individuals)/float64(total_test_iterations)*100.0,
			total_test_iterations, allowed_failure_rate*100.0,
		)
	}

	t.Logf("There are a total of %d validation errors detected when generating university schedules", len(err_list_validation))
}
