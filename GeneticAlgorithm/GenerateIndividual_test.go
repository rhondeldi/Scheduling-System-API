package GeneticAlgorithm_test

import (
	"fmt"
	"log"
	"reflect"
	"slices"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageSchedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func TestEstimateResourceAvailabilityFirstSem(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	departments, err_department := persistence.ReaderService.ReadAllDepartments()

	if err_department != nil {
		t.Fatalf("error loading departments %s", err_department.Error())
	}

	departments = departments[1:]

	log.Printf("Total Departments : %d", len(departments))

	for _, department := range departments {
		missing_resources, err := GeneticAlgorithm.EstimateResourceAvailability(&persistence, GeneticAlgorithm.TERM_1ST_SEMESTER, int(department.DepartmentID))

		if err != nil {
			t.Fatalf("error while estimating resources : %s", err.Error())
		}

		t.Log("Missing Resources :\n\n")

		Utils.PrettyPrint(missing_resources)

		t.Log("\n\n")

		if missing_resources.InstructorTimeSlot > 0 {
			t.Errorf("not enough instructors in %s", department.Name)
		}

		if missing_resources.RoomLecTimeSlot > 0 {
			t.Errorf("not enough lecture rooms in %s", department.Name)
		}

		if missing_resources.RoomLabTimeSlot > 0 {
			t.Errorf("not enough laboratory rooms in %s", department.Name)
		}

		if missing_resources.RoomGymTimeSlot > 0 {
			t.Errorf("not enough gym rooms in %s", department.Name)
		}
	}
}

func TestEstimateResourceAvailabilitySecondSem(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	departments, err_department := persistence.ReaderService.ReadAllDepartments()

	if err_department != nil {
		t.Fatalf("error loading departments %s", err_department.Error())
	}

	departments = departments[1:]

	log.Printf("Total Departments : %d", len(departments))

	for _, department := range departments {
		missing_resources, err := GeneticAlgorithm.EstimateResourceAvailability(&persistence, GeneticAlgorithm.TERM_2ND_SEMESTER, int(department.DepartmentID))

		if err != nil {
			t.Fatalf("error while estimating resources : %s", err.Error())
		}

		t.Log("Missing Resources :\n\n")

		Utils.PrettyPrint(missing_resources)

		t.Log("\n\n")

		if missing_resources.InstructorTimeSlot > 0 {
			t.Fatalf("not enough instructors in %s", department.Name)
		}

		if missing_resources.RoomLecTimeSlot > 0 {
			t.Fatalf("not enough lecture rooms in %s", department.Name)
		}

		if missing_resources.RoomLabTimeSlot > 0 {
			t.Fatalf("not enough laboratory rooms in %s", department.Name)
		}

		if missing_resources.RoomGymTimeSlot > 0 {
			t.Fatalf("not enough gym rooms in %s", department.Name)
		}
	}
}

func TestNewPopulationFirstSem(t *testing.T) {
	total_sections := GeneratePopulations(t, GeneticAlgorithm.TERM_1ST_SEMESTER)

	t.Log("Total Sections For First Semester : ", total_sections)

	if total_sections != 192 {
		t.Fatal("total sections generated is not equal to the expected number of sections")
	}

}

func TestNewPopulationSecondSem(t *testing.T) {
	total_sections := GeneratePopulations(t, GeneticAlgorithm.TERM_2ND_SEMESTER)

	t.Log("Total Sections For Second Semester : ", total_sections)

	if total_sections != 184 {
		t.Fatal("total sections generated is not equal to the expected number of sections")
	}

}

func GeneratePopulations(t *testing.T, target_semester int) int {
	persistence := StorageResources.Persistence{
		ReaderService: &StorageResources.JsonReader{},
		WriterService: &StorageResources.JsonWriter{},
	}

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

	var final_uni_sched Schedule.UniTimeTables

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

			errs_encoding_validation := GeneticAlgorithm.ValidateEncodingResource(university_schedules, encoding_resource, curriculums, target_semester)

			if errs_encoding_validation != nil {
				t.Fatal(errs_encoding_validation)
			}
		}

		final_uni_sched = university_schedules

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

	return len(final_uni_sched)
}

//////////////////////////////////////
// WITH DEPARTMENT SPECIFIC ENCODING
/////////////////////////////////////

func TestNewPopulation1stSemWithDepartmentSelection(t *testing.T) {
	GeneratePopWithDepartmentSelection(t, GeneticAlgorithm.TERM_1ST_SEMESTER)
}

func TestNewPopulation2ndSemWithDepartmentSelection(t *testing.T) {
	GeneratePopWithDepartmentSelection(t, GeneticAlgorithm.TERM_2ND_SEMESTER)
}

func GeneratePopWithDepartmentSelection(t *testing.T, target_semester int) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	t.Logf("Semester : %d\n\n", target_semester)

	total_test_iterations := 256
	allowed_failure_rate := 0.8 // 80% error rate allowed.

	err_list_generation := make([]error, 0, 8)
	err_list_validation := make([]error, 0, 8)

	////////////////////////////////////////////////////////////////////////////////////////

	departments, err_all_departments := persistence.ReaderService.ReadAllDepartments()

	if err_all_departments != nil {
		t.Fatal(err_all_departments)
	}

	rooms, err_all_rooms := persistence.ReaderService.ReadAllRooms()

	if err_all_rooms != nil {
		t.Fatal(err_all_rooms)
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

new_population_loop:
	for i := range total_test_iterations {
		if (i == 0) || (((i + 1) % 16) == 0) {
			fmt.Printf("Generating university schedules [per-department] (%d)..................................\n", (i + 1))
		}

		total_sections_calculated := Curriculum.GetTotalNumberOfSections(curriculums, target_semester)
		empty_university_schedule := GeneticAlgorithm.NewEmptyIndividual(curriculums, target_semester)

		if total_sections_calculated != len(empty_university_schedule) {
			t.Fatalf(
				"the calculated total sections for the semester index %d is %d, but the generated empty schedules only contains %d which is a mismatch",
				target_semester, total_sections_calculated, len(empty_university_schedule),
			)
		}

		all_departments, err_read_all_departments := persistence.ReaderService.ReadAllDepartments()

		if err_read_all_departments != nil {
			t.Fatal(err_read_all_departments)
		}

		is_department_id_to_has_curriculum := make(map[uint16]bool)

		for _, curriculum := range curriculums {
			is_department_id_to_has_curriculum[curriculum.DepartmentID] = true
		}

		track_resources, err_der_copy := default_encoding_resource.MakeCopy()

		if err_der_copy != nil {
			t.Fatal(err_der_copy)
		}

		track_schedules := empty_university_schedule

		if !track_schedules.IsEmpty() {
			t.Fatal("returned not empty university schedule : before loop")
		}

		all_departments = all_departments[1:]

		for department_idx, department := range all_departments {

			// setup specific department for schedule generation

			if has_curriculum := is_department_id_to_has_curriculum[department.DepartmentID]; !has_curriculum {
				continue // skip departments that don't have curriculums yet
			}

			department_to_encode := make(map[uint16]bool)
			department_to_encode[department.DepartmentID] = true

			if department_idx != 0 && track_schedules.IsEmpty() {
				t.Fatalf("returned an empty university schedule : loop iteration %d\n", i)
			}

			var err_copy_resource error
			var err_not_enough_resource error

			retries := 0
			max_retries := 7

			// start generating specific department schedule

			for {
				err_copy_resource = nil
				err_not_enough_resource = nil

				output_schedules, output_resources, err_encode_individual_genome := GeneticAlgorithm.EncodeIndividualGenome(
					track_schedules,
					curriculums, dept_id_to_department,
					track_resources, department_to_encode,
					target_semester, 0,
				)

				if output_schedules == nil && output_resources == nil && err_encode_individual_genome != nil {
					err_copy_resource = err_encode_individual_genome
				} else if output_schedules != nil && output_resources == nil && err_encode_individual_genome != nil {
					err_not_enough_resource = err_encode_individual_genome
				}

				if err_copy_resource != nil {
					t.Fatal(err_copy_resource)
				}

				if err_not_enough_resource != nil {
					retries++

					if retries > max_retries {
						t.Logf("failed to generate individual schedule number %d, after %d tries, for %s error %s\n", i, retries, department.Code, err_not_enough_resource.Error())
						err_list_generation = append(err_list_generation, err_not_enough_resource)
						continue new_population_loop
					}

					t.Logf("retry (%d : %s) - %s\n", retries, department.Code, err_not_enough_resource.Error())
					continue
				}

				if errs := GeneticAlgorithm.ValidateEncodingResource(output_schedules, output_resources, curriculums, target_semester); errs != nil {
					t.Logf("currently at department %s, test iteration %d", department.Code, i+1)
					t.Fatal(errs)
				}

				track_schedules = output_schedules
				track_resources = output_resources
				break // department schedule generated - end retry loop
			}

			if len(track_schedules) == 0 {
				t.Fatal("No university schedules generated")
			}

			if track_schedules == nil {
				t.Fatalf("returned a nil university schedule : loop iteration %d\n", i)
			}

			err_vertical_validations := track_schedules.VerticalValidation(rooms)

			for _, e := range err_vertical_validations {
				t.Error(e)
				err_list_validation = append(err_list_validation, e)
			}

			errs_encoding_validation := GeneticAlgorithm.ValidateEncodingResource(track_schedules, track_resources, curriculums, target_semester)

			if errs_encoding_validation != nil {
				t.Fatal(errs_encoding_validation)
			}

			///////////////////////

			// check generated schedule's encoding resource by generating from the previous generated uni time table and comparing
			// it to the resulting encoding resource from the same previous generated department schedule uni time table

			generated_encoding_resource, output_resources := GeneticAlgorithm.GenerateEncodingResourceFromUniTimeTable(
				track_schedules, curriculums, target_semester, default_encoding_resource,
			)

			if output_resources != nil {
				t.Fatal(output_resources)
			}

			if track_resources != nil {
				if !GeneticAlgorithm.IsEqualEncodingResource(generated_encoding_resource, track_resources) {
					t.Fatal("generated encoding resource from bare university schedule is not equal to the produced encoding resource of GA")
				}
			}

			/////////////////

			if department_idx < len(all_departments)-1 {

				// test horizontal validation for the whole university schedule - there should be an error

				fmt.Printf("Generated schedules for all departments, the department %s\n", department.Name)

				err_intentional_horizontal_validation := GeneticAlgorithm.HorizontalValidation(track_schedules, curriculums, nil, target_semester)

				if err_intentional_horizontal_validation == nil {
					t.Fatal("there should be a missing subject error here since the university schedule is not complete yet")
				}

				// test horizontal validation for the department specific schedule - there should be NO error

				department_to_validate := make(map[uint16]bool)
				department_to_validate[department.DepartmentID] = true

				err_department_horizontal_validations := GeneticAlgorithm.HorizontalValidation(
					track_schedules, curriculums, department_to_validate, target_semester,
				)

				for _, e := range err_department_horizontal_validations {
					t.Fatal(e)
				}
			} else {

				// at the very last department, do test horizontal validation for the whole university schedule - there should be NO error

				fmt.Printf("Generated schedules for all departments, the last department schedules generated is %s\n", department.Name)

				if track_schedules.IsEmpty() {
					t.Fatalf("returned an empty university schedule : loop iteration %d\n", i)
				}

				err_horizontal_validations := GeneticAlgorithm.HorizontalValidation(
					track_schedules, curriculums, nil, target_semester,
				)

				for _, e := range err_horizontal_validations {
					t.Fatal(e)
				}
			}
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

////////////////////////////////
// BENCHMARK
/////////////////////////////////

func BenchmarkNewPopulationFirstSem(b *testing.B) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	////////////////////////////////////////////////////////////////////////////////////////

	departments, err_all_departments := persistence.ReaderService.ReadAllDepartments()

	if err_all_departments != nil {
		b.Fatal(err_all_departments)
	}

	curriculums, err_all_curriculums := persistence.ReaderService.ReadAllCurriculum()

	if err_all_curriculums != nil {
		b.Fatal(err_all_curriculums)
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	////////////////////////////////////////////////////////////////////////////////////////

	default_encoding_resource, err_read_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(&persistence)

	if err_read_default_encoding_resource != nil {
		b.Fatal(err_read_default_encoding_resource)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	for i := 0; i < b.N; i++ {
		empty_university_schedule := GeneticAlgorithm.NewEmptyIndividual(curriculums, GeneticAlgorithm.TERM_1ST_SEMESTER)

		GeneticAlgorithm.EncodeIndividualGenome(
			empty_university_schedule,
			curriculums, dept_id_to_department,
			default_encoding_resource, nil,
			GeneticAlgorithm.TERM_1ST_SEMESTER, 0,
		)
	}
}

func BenchmarkNewPopulationSecondSem(b *testing.B) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	////////////////////////////////////////////////////////////////////////////////////////

	departments, err_all_departments := persistence.ReaderService.ReadAllDepartments()

	if err_all_departments != nil {
		b.Fatal(err_all_departments)
	}

	curriculums, err_all_curriculums := persistence.ReaderService.ReadAllCurriculum()

	if err_all_curriculums != nil {
		b.Fatal(err_all_curriculums)
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	////////////////////////////////////////////////////////////////////////////////////////

	default_encoding_resource, err_read_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(&persistence)

	if err_read_default_encoding_resource != nil {
		b.Fatal(err_read_default_encoding_resource)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	for i := 0; i < b.N; i++ {
		empty_university_schedule := GeneticAlgorithm.NewEmptyIndividual(curriculums, GeneticAlgorithm.TERM_2ND_SEMESTER)

		GeneticAlgorithm.EncodeIndividualGenome(
			empty_university_schedule,
			curriculums, dept_id_to_department,
			default_encoding_resource, nil,
			GeneticAlgorithm.TERM_2ND_SEMESTER, 0,
		)
	}
}
