package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV2"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageSchedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

const ADDED_RESOURCES_FOR_STRESS_TEST int = 1000

var SessionStore = cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))

type DepartmentGenResult struct {
	Semester         int
	Department       Departments.Department
	GenerationResult RouteGlobals.SchedGenResult
	TotalSections    int
}

func TestIntegrationEditCurriculumSectionV1(t *testing.T) {

	// initialize the router

	// setup_router()
	router := setup_router()

	// generate university schedules for all semesters

	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_departments != nil {
		t.Fatalf("Failed to read departments: %v", err_read_departments)
	}

	departments_without_gen := make([]Departments.Department, 0, len(departments))

	for i := range departments {
		if departments[i].DepartmentID == 0 {
			continue
		}

		departments_without_gen = append(departments_without_gen, departments[i])
	}

	departments = departments_without_gen

	for semester := range Curriculum.SUPPORTED_SEMESTERS {
		for _, department := range departments {
			request := httptest.NewRequest(
				http.MethodPost, fmt.Sprintf(
					"/v1/generate_schedule?semester=%d&department_id=%d",
					semester, department.DepartmentID,
				),
				http.NoBody,
			)

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				t.Fatalf("Failed to generate schedules: status code %d, body: %s", response.Code, response.Body.String())
			}
		}
	}

	// wait for the schedules to be generated

	for {
		time.Sleep(15 * time.Second)
		t.Log("Waiting for schedule generation to finish...")

		request := httptest.NewRequest(http.MethodGet, "/v1/gen_status", nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
			t.Fatalf("Failed to check generation status: status code %d, body: %s", response.Code, response.Body.String())
		}

		var get_body struct {
			IsGenerating bool `json:"status"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &get_body); err != nil {
			t.Fatalf("Failed to parse generation status response: %v", err)
		}

		if !get_body.IsGenerating {
			break
		}
	}

	// validate each departments using schedule generation results

	for semester := range Curriculum.SUPPORTED_SEMESTERS {
		for _, department := range departments {
			t.Logf("validating department %s %s", department.Name, Curriculum.SEMESTER_INDEX_NAME[semester])

			///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

			request := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/v1/dept_gen_result?semester=%d&department_id=%d", semester, department.DepartmentID),
				nil,
			)

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				t.Fatalf("Unexpected status code %d for department %d: body: %s", response.Code, department.DepartmentID, response.Body.String())
			} else {
				response_body := &RouteGlobals.SchedGenResult{}
				if err := json.Unmarshal(response.Body.Bytes(), &response_body); err != nil {
					t.Fatalf("Failed to parse generation status response: %v", err)
				}

				if !(semester == 2 && Utils.HasSubString(response_body.Message, "empty")) {
					if response_body.Status != RouteGlobals.SchedGenStatusSuccess {
						t.Fatalf("Failed to generation schedule in %s, %s - %s", department.Code, response_body.Status, response_body.Message)
					}
				}

				t.Logf("Validation result %s : %s", response_body.Status, response_body.Message)
			}
		}
	}

	// validate each department using the API

	for semester := range Curriculum.SUPPORTED_SEMESTERS {
		for _, department := range departments {

			///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

			request := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/v2/validate_schedules?semester=%d&department_id=%d", semester, department.DepartmentID),
				nil,
			)

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			switch response.Code {
			case http.StatusNotFound, http.StatusConflict:
				var validationResponse []string
				if err := json.Unmarshal(response.Body.Bytes(), &validationResponse); err != nil {
					t.Fatalf("Failed to parse validation response for department %d: %v", department.DepartmentID, err)
				}
				if len(validationResponse) > 0 {
					t.Fatalf("Validation errors for department %d: %v", department.DepartmentID, validationResponse)
				}
			default:
				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					t.Fatalf("Unexpected status code %d for department %d: body: %s", response.Code, department.DepartmentID, response.Body.String())
				}
			}
		}
	}

	// edit curriculum section counts

	curriculums, err_read_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_curriculums != nil {
		t.Fatalf("Failed to read curriculums: %v", err_read_curriculums)
	}

	for c, curriculum := range curriculums {
		for y, year_level := range curriculum.YearLevels {
			for s := range year_level.Semesters {
				curriculums[c].YearLevels[y].Semesters[s].Sections = Utils.RandomInRange(1, 6)
			}
		}
	}

	for _, curriculum := range curriculums {

		json_curriculum, err := json.Marshal(curriculum)

		if err != nil {
			panic(err)
		}

		request := httptest.NewRequest(http.MethodPatch, "/v1/curriculum_update", bytes.NewBuffer(json_curriculum))

		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
			t.Fatalf("Failed to edit curriculum sections: status code %d, body: %s", response.Code, response.Body.String())
		}
	}

	// validate the schedules

	t.Log("Validating...")

	for semester := range Curriculum.SUPPORTED_SEMESTERS {
		university_schedule, err_load_sched := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

		if err_load_sched != nil {
			t.Fatalf("Failed to load schedules: %v", err_load_sched)
		}

		GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedule, curriculums, semester, nil, nil,
			func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {

				subject_from_schedule := make(map[uint16]int)
				for _, subject := range university_schedule[indicies.Usi].GetWeekSubjectsJSON() {
					_, has_id := subject_from_schedule[subject.SubjectID]
					if !has_id {
						subject_from_schedule[subject.SubjectID] = subject.TimeSlotSize
					} else {
						subject_from_schedule[subject.SubjectID] += subject.TimeSlotSize
					}
				}

				subject_from_curriculum := make(map[uint16]int)
				for _, subject := range values.Semester.Subjects {
					subject_from_curriculum[subject.ID] = int(subject.LecHours+subject.LabHours) * Const.N_HOUR_TIME_SLOTS
				}

				if (len(subject_from_schedule) != len(subject_from_curriculum)) && indicies.Section < 4 {

					log.Print("from schedule :\n\n")

					Utils.PrettyPrint(subject_from_schedule)

					log.Print("\n\nfrom curriculum :\n\n")

					Utils.PrettyPrint(subject_from_curriculum)

					log.Print("\n\n")

					t.Fatalf(
						"Mismatch in number of subjects, %s, %s, %s, section %s : schedule has %d, curriculum has %d",
						values.Curriculum.CurriculumCode,
						values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
						Curriculum.SECTION[indicies.Section],
						len(subject_from_schedule),
						len(subject_from_curriculum),
					)
				}

				for k, v := range subject_from_schedule {
					if subject_from_schedule[k] != subject_from_curriculum[k] && indicies.Section < 4 {
						t.Fatalf(
							"Mismatch in %s, %s, %s, section %s, subject id %d: schedule has %d time slots, while curriculum has %d time slots",
							values.Curriculum.CurriculumCode,
							values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
							Curriculum.SEMESTER_INDEX_NAME[indicies.Section],
							k, v, subject_from_curriculum[k],
						)
					}
				}

				if indicies.Section >= 4 && len(subject_from_schedule) != 0 {
					t.Fatalf(
						"%s, %s, %s, section %s has %d subjects even though it should not contain any because it is a new section",
						values.Curriculum.CurriculumCode,
						values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
						Curriculum.SECTION[indicies.Section],
						subject_from_schedule,
					)
				} else if indicies.Section < 4 && len(subject_from_schedule) == 0 {
					t.Fatalf(
						"%s, %s, %s, section %s has 0 subjects even though it should contain at least one",
						values.Curriculum.CurriculumCode,
						values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
						Curriculum.SECTION[indicies.Section],
					)
				}

				return GeneticAlgorithm.IterProceed
			},
		)
	}
}

func TestIntegrationEditCurriculumSectionV2(t *testing.T) {
	for test_iteration := range 10 {
		router := setup_router()

		// generate university schedules for all semesters

		departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

		if err_read_departments != nil {
			t.Fatalf("Failed to read departments: %v", err_read_departments)
		}

		departments_without_gen := make([]Departments.Department, 0, len(departments))

		for i := range departments {
			if departments[i].DepartmentID == 0 {
				continue
			}

			departments_without_gen = append(departments_without_gen, departments[i])
		}

		departments = departments_without_gen

		// generate schedules

		for semester := range Curriculum.SUPPORTED_SEMESTERS {
			for _, department := range departments {

				///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

				t.Logf("generating schedule for %s %s", department.Name, Curriculum.SEMESTER_INDEX_NAME[semester])

				request := httptest.NewRequest(
					http.MethodPost, fmt.Sprintf(
						"/v1/generate_schedule?semester=%d&department_id=%d",
						semester, department.DepartmentID,
					),
					http.NoBody,
				)

				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					t.Fatalf("Failed to generate schedules: status code %d, body: %s", response.Code, response.Body.String())
				}
			}
		}

		// wait for the schedules to be generated

		for {
			time.Sleep(15 * time.Second)
			t.Log("Waiting for schedule generation to finish...")

			request := httptest.NewRequest(http.MethodGet, "/v1/gen_status", nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				t.Fatalf("Failed to check generation status: status code %d, body: %s", response.Code, response.Body.String())
			}

			var get_body struct {
				IsGenerating bool `json:"status"`
			}

			if err := json.Unmarshal(response.Body.Bytes(), &get_body); err != nil {
				t.Fatalf("Failed to parse generation status response: %v", err)
			}

			if !get_body.IsGenerating {
				break
			}
		}

		// validate each departments using schedule generation results

		for semester := range Curriculum.SUPPORTED_SEMESTERS {
			for _, department := range departments {

				///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

				request := httptest.NewRequest(
					http.MethodGet,
					fmt.Sprintf("/v1/dept_gen_result?semester=%d&department_id=%d", semester, department.DepartmentID),
					nil,
				)

				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					t.Fatalf("Unexpected status code %d for department %d: body: %s", response.Code, department.DepartmentID, response.Body.String())
				} else {
					response_body := &RouteGlobals.SchedGenResult{}
					if err := json.Unmarshal(response.Body.Bytes(), &response_body); err != nil {
						t.Fatalf("Failed to parse generation status response: %v", err)
					}

					if !(semester == 2 && Utils.HasSubString(response_body.Message, "empty")) {
						if response_body.Status != RouteGlobals.SchedGenStatusSuccess {
							t.Fatalf("Failed to generation schedule in %s, %s - %s", department.Code, response_body.Status, response_body.Message)
						}

						t.Logf("Validation %s result %s : %s", department.Code, response_body.Status, response_body.Message)
					}
				}
			}
		}

		// validate each department using the API

		for semester := range Curriculum.SUPPORTED_SEMESTERS {
			for _, department := range departments {

				///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

				request := httptest.NewRequest(
					http.MethodGet,
					fmt.Sprintf("/v2/validate_schedules?semester=%d&department_id=%d", semester, department.DepartmentID),
					nil,
				)

				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				switch response.Code {
				case http.StatusNotFound, http.StatusConflict:
					var validationResponse []string
					if err := json.Unmarshal(response.Body.Bytes(), &validationResponse); err != nil {
						t.Fatalf("Failed to parse validation response for department %d: %v", department.DepartmentID, err)
					}

					if len(validationResponse) > 0 {
						t.Fatalf("Validation errors for department %d: %v", department.DepartmentID, validationResponse)
					}
				default:
					if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
						t.Fatalf("Unexpected status code %d for department %d: body: %s", response.Code, department.DepartmentID, response.Body.String())
					}
				}
			}
		}

		// edit curriculum section counts

		old_curriculums, err_read_old_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

		if err_read_old_curriculums != nil {
			t.Fatalf("Failed to read old curriculums: %v", err_read_old_curriculums)
		}

		set_new_curriculums, err_read_set_new_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

		if err_read_set_new_curriculums != nil {
			t.Fatalf("Failed to read new curriculums: %v", err_read_set_new_curriculums)
		}

		for c, curriculum := range set_new_curriculums {
			for y, year_level := range curriculum.YearLevels {
				for s := range year_level.Semesters {
					set_new_curriculums[c].YearLevels[y].Semesters[s].Sections = Utils.RandomInRange(1, 5)
				}
			}
		}

		for _, curriculum := range set_new_curriculums {
			json_curriculum, err := json.Marshal(curriculum)

			if err != nil {
				panic(err)
			}

			request := httptest.NewRequest(http.MethodPatch, "/v1/curriculum_update", bytes.NewBuffer(json_curriculum))

			request.Header.Set("Content-Type", "application/json")

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				t.Fatalf("Failed to edit curriculum sections: status code %d, body: %s", response.Code, response.Body.String())
			}
		}

		time.Sleep(30 * time.Second)

		// validate the schedule

		for semester := range Curriculum.SUPPORTED_SEMESTERS {
			university_schedule, err_load_sched := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

			log.Println("university schedule length : ", len(university_schedule))

			if err_load_sched != nil {
				t.Fatalf("Failed to load schedules: %v", err_load_sched)
			}

			new_curriculums, err_read_new_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

			if err_read_new_curriculums != nil {
				t.Fatalf("Failed to read new curriculums: %v", err_read_old_curriculums)
			}

			if !reflect.DeepEqual(new_curriculums, set_new_curriculums) {
				diff := cmp.Diff(new_curriculums, set_new_curriculums)
				t.Fatalf("The curriculums are not equal. Differences:\n%s", diff)
			}

			GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedule, new_curriculums, semester, nil, nil,
				func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {

					if !reflect.DeepEqual(
						university_schedule[indicies.Usi].GetWeekSubjectsJSON(),
						values.WeekSched.GetWeekSubjectsJSON(),
					) {
						t.Fatal("university index schedule subject json not equal to the values.week schedule subject json")
					}

					subject_id_to_timeslots_from_schedule := make(map[uint16]int)
					for _, subject := range university_schedule[indicies.Usi].GetWeekSubjectsJSON() {

						if subject.SubjectID != 0 && subject.TimeSlotSize == 0 {
							t.Fatal("Get Week Subjects JSON has a bug")
						}

						_, has_id := subject_id_to_timeslots_from_schedule[subject.SubjectID]
						if !has_id {
							subject_id_to_timeslots_from_schedule[subject.SubjectID] = subject.TimeSlotSize
						} else {
							subject_id_to_timeslots_from_schedule[subject.SubjectID] += subject.TimeSlotSize
						}
					}

					subject_id_to_timeslots_from_curriculum := make(map[uint16]int)
					for _, subject := range values.Semester.Subjects {
						subject_id_to_timeslots_from_curriculum[subject.ID] = int(subject.LecHours+subject.LabHours) * Const.N_HOUR_TIME_SLOTS
					}

					old_curriculum_sections := old_curriculums[indicies.Curriculum].YearLevels[indicies.YearLevel].Semesters[indicies.Semester].Sections
					new_curriculum_sections := new_curriculums[indicies.Curriculum].YearLevels[indicies.YearLevel].Semesters[indicies.Semester].Sections

					if (len(subject_id_to_timeslots_from_schedule) != len(subject_id_to_timeslots_from_curriculum)) && indicies.Section < old_curriculum_sections {

						log.Print("from schedule :\n\n")

						Utils.PrettyPrint(subject_id_to_timeslots_from_schedule)

						log.Print("\n\nfrom curriculum :\n\n")

						Utils.PrettyPrint(subject_id_to_timeslots_from_curriculum)

						log.Print("\n\n")

						has_subject := false

						for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
							for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
								if university_schedule[indicies.Usi][day][time_slot].GetSubjectID() != 0 {
									has_subject = true
								}
							}
						}

						if has_subject && (len(subject_id_to_timeslots_from_schedule) == 0) {
							t.Logf("")

							t.Fatalf(
								"week schedule has subject but subject from json schedule has length of 0, %s, %s, section %s",
								values.Curriculum.CurriculumCode,
								Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
								Curriculum.SECTION[indicies.Section],
							)
						}

						t.Fatalf(
							"[test-iteration=%d] : Mismatch in number of subjects, %s, %s, %s, section %s : schedule has %d, curriculum has %d, old curriculum sections %d, new curriculum sections %d",
							test_iteration,
							values.Curriculum.CurriculumCode,
							values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
							Curriculum.SECTION[indicies.Section],
							len(subject_id_to_timeslots_from_schedule),
							len(subject_id_to_timeslots_from_curriculum),
							old_curriculum_sections, new_curriculum_sections,
						)
					}

					for k, v := range subject_id_to_timeslots_from_schedule {
						if subject_id_to_timeslots_from_schedule[k] != subject_id_to_timeslots_from_curriculum[k] && indicies.Section < old_curriculum_sections {
							t.Fatalf(
								"[test-iteration=%d-nsdyi] : Mismatch in %s, %s, %s, section %s, subject id %d, schedule has %d time slots, while curriculum has %d time slots, old curriculum sections %d, new curriculum sections %d",
								test_iteration,
								values.Curriculum.CurriculumCode,
								values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
								Curriculum.SECTION[indicies.Section],
								k, v, subject_id_to_timeslots_from_curriculum[k],
								old_curriculum_sections, new_curriculum_sections,
							)
						}
					}

					if indicies.Section >= old_curriculum_sections && len(subject_id_to_timeslots_from_schedule) != 0 {
						t.Fatalf(
							"[test-iteration=%d] : %s, %s, %s, section %s has %d subjects even though it should not contain any because it is a new section, old curriculum sections %d, new curriculum sections %d",
							test_iteration,
							values.Curriculum.CurriculumCode,
							values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
							Curriculum.SECTION[indicies.Section],
							subject_id_to_timeslots_from_schedule,
							old_curriculum_sections, new_curriculum_sections,
						)
					} else if indicies.Section < old_curriculum_sections && len(subject_id_to_timeslots_from_schedule) == 0 {
						t.Fatalf(
							"[test-iteration=%d] : %s, %s, %s, section %s has 0 subjects even though it should contain at least one, old curriculum sections %d, new curriculum sections %d",
							test_iteration,
							values.Curriculum.CurriculumCode,
							values.YearLevel.Name, Curriculum.SEMESTER_INDEX_NAME[indicies.Semester],
							Curriculum.SECTION[indicies.Section],
							old_curriculum_sections, new_curriculum_sections,
						)
					}

					return GeneticAlgorithm.IterProceed
				},
			)
		}
	}
}

func BenchmarkIntegrationTest(b *testing.B) {

	supported_semesters := make([]int, 0)
	supported_semesters = append(supported_semesters, GeneticAlgorithm.TERM_1ST_SEMESTER)
	supported_semesters = append(supported_semesters, GeneticAlgorithm.TERM_2ND_SEMESTER)
	supported_semesters = append(supported_semesters, GeneticAlgorithm.TERM_MIDYEAR)

	router := setup_router()

	// read all curriculums

	final_curriculums, err_read_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_curriculums != nil {
		b.Fatalf("Failed to read new curriculums: %v", err_read_curriculums)
	}

	// generate university schedules for all semesters

	fmt.Println("read departments")

	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_departments != nil {
		b.Fatalf("Failed to read departments: %v", err_read_departments)
	}

	departments_without_gen := make([]Departments.Department, 0, len(departments))

	for i := range departments {
		if departments[i].DepartmentID == 0 {
			continue
		}

		departments_without_gen = append(departments_without_gen, departments[i])
	}

	departments = departments_without_gen

	// generate schedules

	fmt.Println("initial generation - generate university schedules")

	for _, semester := range supported_semesters {
		for _, department := range departments {

			///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

			b.Logf("generating schedule for %s %s", department.Name, Curriculum.SEMESTER_INDEX_NAME[semester])

			request := httptest.NewRequest(
				http.MethodPost, fmt.Sprintf(
					"/v1/generate_schedule?semester=%d&department_id=%d",
					semester, department.DepartmentID,
				),
				http.NoBody,
			)

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				b.Fatalf("Failed to generate schedules: status code %d, body: %s", response.Code, response.Body.String())
			}
		}
	}

	// wait for the schedules to be generated

	fmt.Println("wait...")

	for {
		time.Sleep(30 * time.Second)
		b.Log("Waiting for schedule generation to finish...")

		request := httptest.NewRequest(http.MethodGet, "/v1/gen_status", nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
			b.Fatalf("Failed to check generation status: status code %d, body: %s", response.Code, response.Body.String())
		}

		var get_body struct {
			IsGenerating bool `json:"status"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &get_body); err != nil {
			b.Fatalf("Failed to parse generation status response: %v", err)
		}

		if !get_body.IsGenerating {
			break
		}
	}

	// generate schedules again but this time clear the schedule for each departments first

	fmt.Println("final generation - generate university scehdules")

	for _, semester := range supported_semesters {
		for _, department := range departments {

			{
				b.Logf("clearing schedule for %s %s", department.Name, Curriculum.SEMESTER_INDEX_NAME[semester])

				request := httptest.NewRequest(
					http.MethodDelete, fmt.Sprintf(
						"/v1/clear_department_schedules?semester=%d&department_id=%d",
						semester, department.DepartmentID,
					),
					http.NoBody,
				)

				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					b.Fatalf("Failed to generate schedules: status code %d, body: %s", response.Code, response.Body.String())
				}
			}

			fmt.Println("wait...")

			for {
				time.Sleep(5 * time.Second)
				b.Log("Waiting for schedule generation to finish...")

				request := httptest.NewRequest(http.MethodGet, "/v1/gen_status", nil)
				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					b.Fatalf("Failed to check generation status: status code %d, body: %s", response.Code, response.Body.String())
				}

				var get_body struct {
					IsGenerating bool `json:"status"`
				}

				if err := json.Unmarshal(response.Body.Bytes(), &get_body); err != nil {
					b.Fatalf("Failed to parse generation status response: %v", err)
				}

				if !get_body.IsGenerating {
					break
				}
			}

			{
				///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

				b.Logf("generating schedule for %s %s", department.Name, Curriculum.SEMESTER_INDEX_NAME[semester])

				request := httptest.NewRequest(
					http.MethodPost, fmt.Sprintf(
						"/v1/generate_schedule?semester=%d&department_id=%d",
						semester, department.DepartmentID,
					),
					http.NoBody,
				)

				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					b.Fatalf("Failed to generate schedules: status code %d, body: %s", response.Code, response.Body.String())
				}
			}

			// wait for the schedules to be generated

			for {
				time.Sleep(15 * time.Second)
				b.Log("Waiting for schedule generation to finish...")

				request := httptest.NewRequest(http.MethodGet, "/v1/gen_status", nil)
				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					b.Fatalf("Failed to check generation status: status code %d, body: %s", response.Code, response.Body.String())
				}

				var get_body struct {
					IsGenerating bool `json:"status"`
				}

				if err := json.Unmarshal(response.Body.Bytes(), &get_body); err != nil {
					b.Fatalf("Failed to parse generation status response: %v", err)
				}

				if !get_body.IsGenerating {
					break
				}
			}
		}
	}

	// validate each departments using schedule generation results

	responses_gen := make([]DepartmentGenResult, 0)

	for _, semester := range supported_semesters {
		for _, department := range departments {

			///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

			request := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/v1/dept_gen_result?semester=%d&department_id=%d", semester, department.DepartmentID),
				nil,
			)

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
				b.Fatalf("Unexpected status code %d for department %d: body: %s", response.Code, department.DepartmentID, response.Body.String())
			} else {
				response_body := &RouteGlobals.SchedGenResult{}
				if err := json.Unmarshal(response.Body.Bytes(), &response_body); err != nil {
					b.Fatalf("Failed to parse generation status response: %v", err)
				}

				if !(semester == 2 && Utils.HasSubString(response_body.Message, "empty")) {
					if response_body.Status != RouteGlobals.SchedGenStatusSuccess {
						b.Fatalf("Failed to generation schedule in %s, %s - %s", department.Code, response_body.Status, response_body.Message)
					}

					b.Logf("Validation %s result %s : %s", department.Code, response_body.Status, response_body.Message)
				}

				department_sections := 0

				for _, curr := range final_curriculums {
					if department.DepartmentID == curr.DepartmentID {
						for _, yrlvl := range curr.YearLevels {
							for sem_idx, sem := range yrlvl.Semesters {

								if semester == sem_idx {
									department_sections += sem.Sections
								}
							}
						}
					}
				}

				responses_gen = append(responses_gen, DepartmentGenResult{
					Semester:         semester,
					Department:       department,
					GenerationResult: *response_body,
					TotalSections:    department_sections,
				})
			}
		}
	}

	// validate each department using the API

	for _, semester := range supported_semesters {
		for _, department := range departments {

			///////////////////////////////////////////////// lies here a memory you don't want to remember... ///////////////////

			request := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/v2/validate_schedules?semester=%d&department_id=%d", semester, department.DepartmentID),
				nil,
			)

			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			switch response.Code {
			case http.StatusNotFound, http.StatusConflict:
				var validationResponse []string
				if err := json.Unmarshal(response.Body.Bytes(), &validationResponse); err != nil {
					b.Fatalf("Failed to parse validation response for department %d: %v", department.DepartmentID, err)
				}

				if len(validationResponse) > 0 {
					b.Fatalf("Validation errors for department %d: %v", department.DepartmentID, validationResponse)
				}
			default:
				if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
					b.Fatalf("Unexpected status code %d for department %d: body: %s", response.Code, department.DepartmentID, response.Body.String())
				}
			}
		}
	}

	total_sections := make(map[uint16]map[int]int)

	for _, response_i := range responses_gen {
		fmt.Printf(
			"%s - [%s] took %s - %s\t\tfor %d sections\n",
			response_i.Department.Code,
			response_i.GenerationResult.Status,
			response_i.GenerationResult.Message,
			Curriculum.SEMESTER_INDEX_NAME[response_i.Semester],
			response_i.TotalSections,
		)

		if _, has_department_id := total_sections[response_i.Department.DepartmentID]; !has_department_id {
			total_sections[response_i.Department.DepartmentID] = make(map[int]int)
		}

		if _, has_semester := total_sections[response_i.Department.DepartmentID][response_i.Semester]; !has_semester {
			total_sections[response_i.Department.DepartmentID][response_i.Semester] = 0
		}

		fmt.Printf("\ndepartment fitness progression: %v\n\n", response_i.GenerationResult.FitnessProgressionDepartment)
		fmt.Printf("university fitness progression: %v\n\n", response_i.GenerationResult.FitnessProgressionUniversity)

		total_sections[response_i.Department.DepartmentID][response_i.Semester] += response_i.TotalSections
	}

	fmt.Println()

	for department_id, semester_map := range total_sections {
		for semester_idx := range semester_map {
			fmt.Printf(
				"department id %d - %s => total sections %d\n",
				department_id, Curriculum.SEMESTER_INDEX_NAME[semester_idx],
				total_sections[department_id][semester_idx],
			)
		}
	}
}

func AddResourcesAndSectionForBenchmarkIntegrationStressTest(b *testing.B) {

	supported_semesters := make([]int, 0)
	supported_semesters = append(supported_semesters, GeneticAlgorithm.TERM_1ST_SEMESTER)
	supported_semesters = append(supported_semesters, GeneticAlgorithm.TERM_2ND_SEMESTER)
	supported_semesters = append(supported_semesters, GeneticAlgorithm.TERM_MIDYEAR)

	router := setup_router()

	// edit curriculum section counts

	fmt.Println("edit curriculum section count")

	set_new_curriculums, err_read_set_new_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_set_new_curriculums != nil {
		b.Fatalf("Failed to read new curriculums: %v", err_read_set_new_curriculums)
	}

	semester_idx_to_department_id_to_department_section_count := make(map[int]map[uint16]int)

	for c, curriculum := range set_new_curriculums {
		for y, year_level := range curriculum.YearLevels {
			for s := range year_level.Semesters {

				if !slices.Contains(supported_semesters, s) {
					continue
				}

				set_new_curriculums[c].YearLevels[y].Semesters[s].Sections = 30

				if _, has_semester := semester_idx_to_department_id_to_department_section_count[s]; !has_semester {
					semester_idx_to_department_id_to_department_section_count[s] = make(map[uint16]int)
				}

				if _, has_dept_id := semester_idx_to_department_id_to_department_section_count[s][curriculum.DepartmentID]; !has_dept_id {
					semester_idx_to_department_id_to_department_section_count[s][curriculum.DepartmentID] = set_new_curriculums[c].YearLevels[y].Semesters[s].Sections
				} else {
					semester_idx_to_department_id_to_department_section_count[s][curriculum.DepartmentID] += set_new_curriculums[c].YearLevels[y].Semesters[s].Sections
				}

				// remove all currently designated instructors to avoid error

				for subj_i := range set_new_curriculums[c].YearLevels[y].Semesters[s].Subjects {
					set_new_curriculums[c].YearLevels[y].Semesters[s].Subjects[subj_i].DesignatedInstructors = nil
				}
			}
		}
	}

	// add resources

	fmt.Println("rooms and instructor resources")

	departments, err_read_all_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_all_departments != nil {
		b.Fatal("error reading all departments")
	}

	for _, department := range departments {

		fmt.Printf("adding resources in %s\n", department.Name)

		for i := range ADDED_RESOURCES_FOR_STRESS_TEST {
			err_create_room_LAB := RouteGlobals.ResourcesPersistence.WriterService.CreateRoom(Rooms.Room{
				DepartmentID: department.DepartmentID,
				Capacity:     1,
				RoomType:     Rooms.ROOM_TYPE_LAB,
				Name:         fmt.Sprintf("Dept%dType%dRoomAdd%d", department.DepartmentID, Rooms.ROOM_TYPE_LAB, i),
			})

			if err_create_room_LAB != nil {
				b.Fatal(err_create_room_LAB)
			}

			err_create_room_LEC := RouteGlobals.ResourcesPersistence.WriterService.CreateRoom(Rooms.Room{
				DepartmentID: department.DepartmentID,
				Capacity:     1,
				RoomType:     Rooms.ROOM_TYPE_LEC,
				Name:         fmt.Sprintf("Dept%dType%dRoomAdd%d", department.DepartmentID, Rooms.ROOM_TYPE_LEC, i),
			})

			if err_create_room_LEC != nil {
				b.Fatal(err_create_room_LEC)
			}

			err_create_room_GYM := RouteGlobals.ResourcesPersistence.WriterService.CreateRoom(Rooms.Room{
				DepartmentID: department.DepartmentID,
				Capacity:     1,
				RoomType:     Rooms.ROOM_TYPE_GYM,
				Name:         fmt.Sprintf("Dept%dType%dRoomAdd%d", department.DepartmentID, Rooms.ROOM_TYPE_GYM, i),
			})

			if err_create_room_GYM != nil {
				b.Fatal(err_create_room_GYM)
			}

			err_create_instructor := RouteGlobals.ResourcesPersistence.WriterService.CreateInstructor(Instructors.Instructor{
				DepartmentID:  department.DepartmentID,
				FirstName:     fmt.Sprintf("FirstDept%dInst%d", department.DepartmentID, i),
				MiddleInitial: fmt.Sprintf("MidDept%dInst%d", department.DepartmentID, i),
				LastName:      fmt.Sprintf("LastDept%dInst%d", department.DepartmentID, i),
			})

			if err_create_instructor != nil {
				b.Fatal(err_create_instructor)
			}
		}
	}

	fmt.Println("wait...")
	time.Sleep(120 * time.Second)

	///

	fmt.Println("patch request edited curriculums")

	for _, curriculum := range set_new_curriculums {
		json_curriculum, err := json.Marshal(curriculum)

		if err != nil {
			panic(err)
		}

		request := httptest.NewRequest(http.MethodPatch, "/v1/curriculum_update", bytes.NewBuffer(json_curriculum))

		request.Header.Set("Content-Type", "application/json")

		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
			b.Fatalf("Failed to edit curriculum sections: status code %d, body: %s", response.Code, response.Body.String())
		}
	}

	fmt.Println("wait...")

	for {
		time.Sleep(30 * time.Second)
		b.Log("Waiting for schedule generation to finish...")

		request := httptest.NewRequest(http.MethodGet, "/v1/gen_status", nil)
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
			b.Fatalf("Failed to check generation status: status code %d, body: %s", response.Code, response.Body.String())
		}

		var get_body struct {
			IsGenerating bool `json:"status"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &get_body); err != nil {
			b.Fatalf("Failed to parse generation status response: %v", err)
		}

		if !get_body.IsGenerating {
			break
		}
	}
}

func BenchmarkIntegrationStressTest(b *testing.B) {
	AddResourcesAndSectionForBenchmarkIntegrationStressTest(b)
	BenchmarkIntegrationTest(b)
}

func setup_router() *gin.Engine {

	switch os.Getenv("USE_DATABASE") {

	case "MongoDB":

		//////////////////////////////////////////////////////////////////////////
		//                         MongoDB Persistence
		//////////////////////////////////////////////////////////////////////////

		mongo_client := StorageResources.NewMongodbClient()
		defer StorageResources.CloseMongodbClient(mongo_client)

		RouteGlobals.ResourcesPersistence = &StorageResources.Persistence{
			ReaderService: &StorageResources.MongodbReader{
				Mongo: &StorageResources.MongoDB{Client: mongo_client},
			},

			WriterService: &StorageResources.MongodbWriter{
				Mongo: &StorageResources.MongoDB{Client: mongo_client},
			},
		}

		RouteGlobals.SchedulePersistence = &StorageSchedule.Persistence{
			LoadService: &StorageSchedule.MongodbReader{
				Mongo: &StorageSchedule.MongoDB{Client: mongo_client},
			},

			SaveService: &StorageSchedule.MongodbWriter{
				Mongo: &StorageSchedule.MongoDB{Client: mongo_client},
			},
		}

	default:

		//////////////////////////////////////////////////////////////////////////
		//                          JSON Persistence
		//////////////////////////////////////////////////////////////////////////

		RouteGlobals.ResourcesPersistence = &StorageResources.Persistence{
			ReaderService: &StorageResources.JsonReader{},
			WriterService: &StorageResources.JsonWriter{},
		}

		RouteGlobals.SchedulePersistence = &StorageSchedule.Persistence{
			LoadService: &StorageSchedule.JsonReader{},
			SaveService: &StorageSchedule.JsonWriter{},
		}
	}

	//////////////////////////////////////////////////////////////////////////
	//                         Initialize Globals
	//////////////////////////////////////////////////////////////////////////

	RouteGlobals.InitializeCachedUniversitySchedule()
	RouteGlobals.InitDeptSchedGenQueue()

	//////////////////////////////////////////////////////////////////////////

	var router *gin.Engine

	if gin.Mode() == gin.ReleaseMode {
		router = gin.Default()
	} else {
		gin.SetMode(gin.TestMode)
		router = gin.Default()
	}

	// maximum memory limit for multipart form file uploads
	router.MaxMultipartMemory = 5 << 20 // 5 MiB

	router.Use(static.Serve("/", static.LocalFile("./dist", true)))

	//////////////////////////////////////////////////////////////////////////
	//                              API-v1
	//////////////////////////////////////////////////////////////////////////

	v1 := router.Group("/v1")
	v2 := router.Group("/v2")

	v1.GET("/const", RoutesV1.GetConst)

	// ============= department routes and handlers =============

	v1.GET("/all_departments", RoutesV1.GetAllDepartments)
	v1.GET("/departments", RoutesV1.GetDepartmentsPaginated)
	v1.GET("/department_data", RoutesV1.GetCurriculumsDataInDepartment)
	v1.POST("/department_add", RoutesV1.PostDepartment)
	v1.PATCH("/department_update", RoutesV1.PatchDepartment)
	v1.DELETE("/department_remove", RoutesV1.DeleteDepartment)

	// ============= instructor routes and handlers =============

	v1.POST("/instructor_add", RoutesV1.PostInstructor)
	v1.PATCH("/instructor_update", RoutesV1.PatchInstructor)
	v1.DELETE("/instructor_remove", RoutesV1.DeleteInstructor)

	v2.GET("instructor_basic", RoutesV2.GetInstructorBasic)
	v2.GET("instructors", RoutesV2.GetDepartmentInstructors)
	v2.GET("instructor_resources", RoutesV2.GetInstructorResource)

	// ============= room routes and handlers =============

	v1.GET("/rooms", RoutesV1.GetDepartmentRooms)
	v1.POST("/room_add", RoutesV1.PostRoom)
	v1.PATCH("/room_update", RoutesV1.PatchRoom)
	v1.DELETE("/room_remove", RoutesV1.DeleteRoom)

	// ============= subject routes and handlers =============

	v1.GET("/subjects", RoutesV1.GetSubjects)
	v1.POST("/subject_add", RoutesV1.PostSubject)
	v1.PATCH("/subject_update", RoutesV1.PatchSubject)
	v1.DELETE("/subject_remove", RoutesV1.DeleteSubject)

	// ============= curriculum routes and handlers =============

	v1.GET("/curriculum_list", RoutesV1.GetDepartmentCurriculumList)
	v1.GET("/curriculum_load", RoutesV1.GetCurriculum)
	v1.POST("/curriculum_add", RoutesV1.PostCurriculum)
	v1.PATCH("/curriculum_update", RoutesV1.PatchCurriculum)
	v1.DELETE("/curriculum_remove", RoutesV1.DeleteCurriculum)

	// ============= schedule routes and handlers =============

	v1.GET("/gen_status", RoutesV1.GetGenStatus)
	v1.GET("/dept_gen_result", RoutesV1.GetDeptartmentGenerationResult)

	v1.GET("/university_schedule", RoutesV1.GetUniversitySchedule)
	v1.POST("/university_schedule", RoutesV1.PostUniversitySchedule)

	v1.GET("/class_schedule", RoutesV1.GetClassSchedule)
	v2.GET("/class_json_schedule", RoutesV2.GetJsonClassSchedule)
	v2.DELETE("/clear_class_schedule", RoutesV2.DeleteClearClassSchedule)
	v1.DELETE("/clear_department_schedules", RoutesV1.DeleteClearDepartmentSchedule)
	v1.POST("/generate_schedule", RoutesV1.RequestGenerateSchedule)

	v2.POST("/available_subject_moves", RoutesV2.GetSubjectAvailableTimeSlotMoves)
	v2.POST("/subject_move", RoutesV2.PostSubjectTimeSlotMove)

	v2.GET("/validate_schedules", RoutesV2.GetValidateSchedules)

	if os.Getenv("GIN_MODE") != "release" {
		v1.GET("/generate_schedule", RoutesV1.RequestGenerateSchedule) // for dev only
	}

	// ============= survery routes and handlers =============

	v2.POST("/add_schedule_preference", RoutesV2.PostWeekTimeTableSurvery)

	return router
}
