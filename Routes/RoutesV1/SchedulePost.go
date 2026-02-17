package RoutesV1

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

const MAX_GENETIC_ALGORITHM_RETRY int = 3
const POPULATION_SIZE = 200
const TOTAL_GENERATION = 32

var request_gen_sched_mutex sync.Mutex

/*
POST:

	"/generate_schedule?semester=[0-1]&department_id=[N>0]"
*/
func RequestGenerateSchedule(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	request_gen_sched_mutex.Lock()
	defer request_gen_sched_mutex.Unlock()

	is_already_generating_schedules := RouteGlobals.IsGeneratingSchedule.Load()
	RouteGlobals.IsGeneratingSchedule.Store(true)

	semester, is_valid_semester_idx := IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_idx {
		RouteGlobals.IsGeneratingSchedule.Store(false)
		return
	}

	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_departments != nil {
		log.Print("RequestGenerateSchedule: [read-departments-error] caused by ", err_read_departments)
		ctx.String(http.StatusInternalServerError, "we're unable to retrieve the departments right now")
		RouteGlobals.IsGeneratingSchedule.Store(false)
		return
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	department_id, is_valid_department_id := IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id {
		RouteGlobals.IsGeneratingSchedule.Store(false)
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, uint16(department_id)); !is_allowed {
		RouteGlobals.IsGeneratingSchedule.Store(false)
		return
	}

	response_msg := ""
	response_status := http.StatusAccepted

	if RouteGlobals.PushNewDeptToDeptSchedGenQueue(RouteGlobals.DeptSchedGenKey{
		DepartmentID: uint16(department_id),
		Semester:     semester,
	}) {
		response_msg += fmt.Sprintf(
			"%s %s was added to the schedule generation queue,",
			dept_id_to_department[uint16(department_id)].Name,
			Curriculum.SEMESTER_INDEX_NAME[semester],
		)

		RouteGlobals.SetDeptSchedGenResult(
			RouteGlobals.DeptSchedGenKey{DepartmentID: uint16(department_id), Semester: semester},
			RouteGlobals.SchedGenResult{
				Status:  RouteGlobals.SchedGenStatusOnQueue,
				Message: "waiting other department schedule generation request to finish",
			},
		)
	} else {
		response_msg += fmt.Sprintf(
			"the %s %s is already in schedule generation queue,",
			dept_id_to_department[uint16(department_id)].Name,
			Curriculum.SEMESTER_INDEX_NAME[semester],
		)

		response_status = http.StatusContinue
	}

	if !is_already_generating_schedules {
		response_msg += " the schedule generation function has started"
		go encode_schedule()
	} else {
		response_msg += " the schedule generation function is already running"
	}

	log.Printf("GenerateSchedule: generating schedule")
	ctx.String(response_status, response_msg)
}

func encode_schedule() {
	RouteGlobals.IsGeneratingSchedule.Store(true)
	defer RouteGlobals.IsGeneratingSchedule.Store(false)

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	log.Println("encode_schedule: [function started]")

	////////////////////////////////////////////////////////////////////////////////////////

	rooms, err_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_rooms != nil {
		log.Fatal("encode_schedule: read room error : ", err_rooms)

		for {
			department_to_encode, semester_to_encode, err_pop_from_queue := RouteGlobals.PopDepartmentToEncodeFromSchedGenQueue()

			if err_pop_from_queue != nil || !has_department_to_encode(department_to_encode) {
				break // no more department and semester in the schedule generation queue to be encoded in the schedules
			}

			for dept_id_key, is_to_encode_dept := range department_to_encode {

				if !is_to_encode_dept {
					continue
				}

				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: dept_id_key, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status:  RouteGlobals.SchedGenStatusInternalError,
						Message: "read room error occur during schedule generation preperation",
					},
				)
			}
		}

		return
	}

	curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_curriculums != nil {
		log.Fatal("encode_schedule: read curriculum error : ", err_curriculums)

		for {
			department_to_encode, semester_to_encode, err_pop_from_queue := RouteGlobals.PopDepartmentToEncodeFromSchedGenQueue()

			if err_pop_from_queue != nil || !has_department_to_encode(department_to_encode) {
				break // no more department and semester in the schedule generation queue to be encoded in the schedules
			}

			for dept_id_key, is_to_encode_dept := range department_to_encode {

				if !is_to_encode_dept {
					continue
				}

				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: dept_id_key, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status:  RouteGlobals.SchedGenStatusInternalError,
						Message: "read curriculum error occur during schedule generation preperation",
					},
				)
			}
		}

		return
	}

	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_departments != nil {
		log.Fatal("encode_schedule: read department error : ", err_read_departments)

		for {
			department_to_encode, semester_to_encode, err_pop_from_queue := RouteGlobals.PopDepartmentToEncodeFromSchedGenQueue()

			if err_pop_from_queue != nil || !has_department_to_encode(department_to_encode) {
				break // no more department and semester in the schedule generation queue to be encoded in the schedules
			}

			for dept_id_key, is_to_encode_dept := range department_to_encode {

				if !is_to_encode_dept {
					continue
				}

				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: dept_id_key, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status:  RouteGlobals.SchedGenStatusInternalError,
						Message: "read department error occur during schedule generation preperation",
					},
				)
			}
		}

		return
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	default_empty_encoding_resource, err_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(RouteGlobals.ResourcesPersistence)

	if err_default_encoding_resource != nil {
		log.Print("encode_schedule: read default encoding resource error : ", err_default_encoding_resource)

		for {
			department_to_encode, semester_to_encode, err_pop_from_queue := RouteGlobals.PopDepartmentToEncodeFromSchedGenQueue()

			if err_pop_from_queue != nil || !has_department_to_encode(department_to_encode) {
				break // no more department and semester in the schedule generation queue to be encoded in the schedules
			}

			for dept_id_key, is_to_encode_dept := range department_to_encode {

				if !is_to_encode_dept {
					continue
				}

				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: dept_id_key, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status:  RouteGlobals.SchedGenStatusInternalError,
						Message: "read default encoding resource error occur during schedule generation preperation",
					},
				)
			}
		}

		return
	}

	////////////////////////////////////////////////////////////////////////////////////////

	var generated_encoding_resource *GeneticAlgorithm.EncodingResource
	var err_gen_encoding_resource error

queue_pop_loop:
	for {
		start := time.Now()

		// get the first department in the queue that requested to generate a schedule for a specific semester

		department_to_encode, semester_to_encode, err_pop_from_queue := RouteGlobals.PopDepartmentToEncodeFromSchedGenQueue()

		if err_pop_from_queue != nil || !has_department_to_encode(department_to_encode) {
			break // no more department and semester in the schedule generation queue to be encoded in the schedules
		}

		// get department id

		department_id := uint16(0)

		for k := range department_to_encode {
			department_id = k
		}

		if department_id <= 0 {
			continue // a sanity check, don't generate schedules for department id that is less than 1
		}

		log.Println("encode_schedule: pop latest task from scedule generation request queue")

		RouteGlobals.SetDeptSchedGenResult(
			RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
			RouteGlobals.SchedGenResult{
				Status:  RouteGlobals.SchedGenStatusInProgress,
				Message: "schedule generation is now in progress",
			},
		)

		// get the current university schedules for the specific semester requested by the first department in the queue

		university_schedule, err_obtain_uni_sched_no_ctx := ObtainUniversityScheduleNoContextNoHorizontalValidation(semester_to_encode)

		if err_obtain_uni_sched_no_ctx != nil {
			RouteGlobals.SetDeptSchedGenResult(
				RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
				RouteGlobals.SchedGenResult{
					Status: RouteGlobals.SchedGenStatusInternalError,
					Message: fmt.Sprintf(
						"error base obtaining university schedule for %s %s, caused by : %s",
						dept_id_to_department[department_id].Name,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						err_gen_encoding_resource.Error(),
					),
				},
			)

			log.Printf(
				"encode_schedule: error obtaining base university schedule for %s %s, caused by : %s",
				dept_id_to_department[department_id].Name,
				Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
				err_gen_encoding_resource.Error(),
			)

			continue
		}

		generated_encoding_resource, err_gen_encoding_resource = GeneticAlgorithm.GenerateEncodingResourceFromUniTimeTable(
			university_schedule, curriculums, semester_to_encode, default_empty_encoding_resource,
		)

		if err_gen_encoding_resource != nil {
			log.Printf(
				"encode_schedule: error generating encoding resource for the base schedule of %s %s, caused by : %s",
				dept_id_to_department[department_id].Name,
				Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
				err_gen_encoding_resource.Error(),
			)

			RouteGlobals.SetDeptSchedGenResult(
				RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
				RouteGlobals.SchedGenResult{
					Status: RouteGlobals.SchedGenStatusInternalError,
					Message: fmt.Sprintf(
						"error generating encoding resource for the base schedule of %s %s, caused by : %s",
						dept_id_to_department[department_id].Name,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						err_gen_encoding_resource.Error(),
					),
				},
			)

			continue
		}

		log.Printf(
			"encode_schedule: generating schedule for %s %s",
			dept_id_to_department[department_id].Code,
			Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
		)

		// record other department's horizontal validation result to compare
		// later after generating schedule for the current department

		is_other_dept_valid_initial := make(map[uint16]bool)

		for other_dept_id := range dept_id_to_department {

			if department_to_encode[other_dept_id] {
				continue
			}

			other_dept_to_validate := make(map[uint16]bool)
			other_dept_to_validate[other_dept_id] = true

			errs_hv := GeneticAlgorithm.HorizontalValidation(
				university_schedule, curriculums, other_dept_to_validate, semester_to_encode,
			)

			is_other_dept_valid_initial[other_dept_id] = len(errs_hv) == 0

			log.Printf(
				"encode_schedule: [other-sched-initial] %s %s no errors: %t",
				dept_id_to_department[other_dept_id].Code,
				Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
				is_other_dept_valid_initial[other_dept_id],
			)
		}

		// encode a new schedule in the obtained university schedule for the specific department

		var fitness_progression_department []float64
		var fitness_progression_university []float64

		var retry int // incremented by the for loop

		for retry = 0; retry < MAX_GENETIC_ALGORITHM_RETRY; retry++ {

			log.Printf(
				"encode_schedule: running genetic algorithm for %s %s (try : %d)",
				dept_id_to_department[department_id].Code,
				Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
				retry+1,
			)

			// generate the encoding resource for the obtained university schedule

			previous_fitness := -50.0

			fitness_progression_department = make([]float64, 0)
			fitness_progression_university = make([]float64, 0)

			fittest_uni_sched, fittest_encoding_resource, err_genetic_algorithm := GeneticAlgorithm.RunGeneticAlgorithm(
				university_schedule, curriculums, rooms, dept_id_to_department,
				default_empty_encoding_resource, generated_encoding_resource,
				department_to_encode, semester_to_encode,
				POPULATION_SIZE, TOTAL_GENERATION,
				RouteGlobals.ResourcesPersistence, func(generation int, generation_fittest_sched Schedule.UniTimeTables, fittest_university_schedule_fitness float64) {

					department_schedule_fitness := GeneticAlgorithm.MeasureUniSchedBasicFitness(
						generation_fittest_sched, curriculums,
						department_to_encode, semester_to_encode,
					)

					fitness_progression_department = append(fitness_progression_department, department_schedule_fitness)
					fitness_progression_university = append(fitness_progression_university, fittest_university_schedule_fitness)

					RouteGlobals.SetDeptSchedGenResult(
						RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
						RouteGlobals.SchedGenResult{
							Status: RouteGlobals.SchedGenStatusInProgress,
							Message: fmt.Sprintf(
								"running genetic algorithm, generation %d/%d, population size %d, department schedule fitness at %f, overall university schedule fitness at %f",
								generation, TOTAL_GENERATION, POPULATION_SIZE, department_schedule_fitness, fittest_university_schedule_fitness,
							),
						},
					)

					// save genetic algorithm's generated in-between university schedule when there's new highest fit schedule

					if department_schedule_fitness <= previous_fitness {
						return
					}

					if err := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(generation_fittest_sched, semester_to_encode); err != nil {
						RouteGlobals.SetDeptSchedGenResult(
							RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
							RouteGlobals.SchedGenResult{
								Status: RouteGlobals.SchedGenStatusInternalError,
								Message: fmt.Sprintf(
									"error saving schedule in between generation after %s, caused by : %s",
									time.Since(start),
									err.Error(),
								),
							},
						)

						log.Print("encode_schedule: [in-between-save-failed] error unable to save the genetic algorithm's generated schedule, caused by :", err.Error())
						return
					} else {

						// cache genetic algorithm's generated university schedule

						if err := RouteGlobals.SetCachedUniversitySchedule(semester_to_encode, generation_fittest_sched); err != nil {
							RouteGlobals.SetDeptSchedGenResult(
								RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
								RouteGlobals.SchedGenResult{
									Status: RouteGlobals.SchedGenStatusInternalError,
									Message: fmt.Sprintf(
										"error caching schedule after %s, schedule was saved but might not reflect right away. caused by %s",
										time.Since(start),
										err.Error(),
									),
								},
							)

							log.Print("encode_schedule: [cache-failed] unable to change the genetic algorithm's generated schedule:", err.Error())
						}

						log.Print("encode_schedule: genetic algorithm's generated schedule cached successfully")
					}
				},
			)

			if err_genetic_algorithm != nil {
				if retry >= MAX_GENETIC_ALGORITHM_RETRY-1 {

					RouteGlobals.SetDeptSchedGenResult(
						RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
						RouteGlobals.SchedGenResult{
							Status:  RouteGlobals.SchedGenStatusFailed,
							Message: err_genetic_algorithm.Error(),
						},
					)

					log.Printf(
						"encode_schedule: [failed] genetic algorithm was unable to generate schedules for %s %s after %d tries, caused by %s",
						dept_id_to_department[department_id].Code,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						retry+1, err_genetic_algorithm.Error(),
					)

					continue queue_pop_loop
				}

				log.Printf(
					"encode_schedule: [retrying] genetic algorithm was unable to generate schedules for %s %s, retrying for %d times..., error caused by:\n\n%s",
					dept_id_to_department[department_id].Code,
					Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					retry+2, err_genetic_algorithm.Error(),
				)

				continue
			}

			log.Printf(
				"encode_schedule: genetic algorithm has generated schedules for %s %s after %d tries",
				dept_id_to_department[department_id].Code,
				Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
				retry+1,
			)

			// check if schedule is nil

			if fittest_uni_sched == nil {
				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status: RouteGlobals.SchedGenStatusFailed,
						Message: fmt.Sprintf(
							"genetic algorithm generated a 'nil' schedule for %s %s after %d tries",
							dept_id_to_department[department_id].Code,
							Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
							retry+1,
						),
					},
				)

				if retry >= MAX_GENETIC_ALGORITHM_RETRY-1 {
					log.Printf(
						"encode_schedule: [failed] genetic algorithm generated a 'nil' schedule for %s %s after %d tries",
						dept_id_to_department[department_id].Code,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						retry+1,
					)

					continue queue_pop_loop
				}

				log.Printf(
					"encode_schedule: [retrying] genetic algorithm generated a 'nil' schedule for %s %s, retrying for %d times...",
					dept_id_to_department[department_id].Code,
					Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					retry+2,
				)

				continue
			}

			// check if schedule is empty

			if GeneticAlgorithm.IsDepartmentScheduleEmpty(fittest_uni_sched, curriculums, semester_to_encode, department_to_encode) {

				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status: RouteGlobals.SchedGenStatusFailed,
						Message: fmt.Sprintf(
							"genetic algorithm generated a 'empty' schedule for %s %s after %d tries",
							dept_id_to_department[department_id].Code,
							Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
							retry+1,
						),
					},
				)

				if retry >= MAX_GENETIC_ALGORITHM_RETRY-1 {
					log.Printf(
						"encode_schedule: [failed] genetic algorithm generated a 'empty' schedule for %s %s after %d tries",
						dept_id_to_department[department_id].Code,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						retry+1,
					)

					continue queue_pop_loop
				}

				log.Printf(
					"encode_schedule: [retrying] genetic algorithm generated a 'empty' schedule for %s %s, retrying for %d times...",
					dept_id_to_department[department_id].Code,
					Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					retry+2,
				)

				continue
			}

			// some sanity checks and log checks

			if reflect.DeepEqual(university_schedule, fittest_uni_sched) {
				log.Print("encode_schedule: (equal) genetic algorithm didn't change the original base schedules")
			} else {
				log.Print("encode_schedule: (not-equal) genetic algorithm generated new schedules")
			}

			if GeneticAlgorithm.IsEqualEncodingResource(generated_encoding_resource, fittest_encoding_resource) {
				log.Print("encode_schedule: (equal) genetic algorithm didn't change the original base encoding resources")
			} else {
				log.Print("encode_schedule: (not-equal) genetic algorithm generated new encoding resources")
			}

			if len(fittest_encoding_resource.DeptIdToInstructors) <= 0 {
				panic("this re-encoding resource has an empty DeptIdToInstructors")
			}

			if len(fittest_encoding_resource.DeptIdToRoomtypeToRooms) <= 0 {
				panic("this re-encoding resource has an empty DeptIdToRoomtypeToRooms")
			}

			if len(fittest_encoding_resource.IsSchedIdxToSubIdToSkip) <= 0 {
				panic("this re-encoding resource has an empty IsSchedIdxToSubIdToSkip")
			}

			// check for overlapping instructors and rooms

			if err := fittest_uni_sched.VerticalValidation(rooms); len(err) > 0 {
				if retry >= MAX_GENETIC_ALGORITHM_RETRY-1 {
					RouteGlobals.SetDeptSchedGenResult(
						RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
						RouteGlobals.SchedGenResult{
							Status:  RouteGlobals.SchedGenStatusFailed,
							Message: "error in genetic algorithm final output schedule, there are overlapping possibly either instructors or rooms in the generated schedule",
						},
					)

					log.Printf(
						"encode_schedule: [failed] error genetic algorithm output schedule has vertical overlaps for %s %s after %d tries.\n\n%v\n\n",
						dept_id_to_department[department_id].Code,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						retry+1, err,
					)

					continue queue_pop_loop
				}

				log.Printf(
					"encode_schedule: [retrying] error genetic algorithm output schedule has vertical overlaps for %s %s, retrying for %d times...\n\n%v\n\n",
					dept_id_to_department[department_id].Code,
					Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					retry+2, err,
				)

				continue
			} else {
				log.Print("encode_schedule: [passed] vertical validation")
			}

			// check for missing subjects or missing subject time slot allocations

			errs_horizontal_validation := GeneticAlgorithm.HorizontalValidation(
				fittest_uni_sched, curriculums, department_to_encode, semester_to_encode,
			)

			if len(errs_horizontal_validation) > 0 {
				if retry >= MAX_GENETIC_ALGORITHM_RETRY-1 {

					RouteGlobals.SetDeptSchedGenResult(
						RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
						RouteGlobals.SchedGenResult{
							Status:  RouteGlobals.SchedGenStatusFailed,
							Message: "error in genetic algorithm final output schedule, there are missing subjects or time slots to the final generated schedule",
						},
					)

					log.Printf(
						"encode_schedule: [failed] error genetic algorithm output schedule has 'horizontal' validation problems for %s %s after %d tries.\n\n%v\n\n",
						dept_id_to_department[department_id].Code,
						Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
						retry+1, errs_horizontal_validation,
					)

					continue queue_pop_loop
				}

				log.Printf(
					"encode_schedule: [retrying] error genetic algorithm output schedule has 'horizontal' validation problems for %s %s, retrying for %d times...\n\n%v\n\n",
					dept_id_to_department[department_id].Code,
					Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					retry+2, errs_horizontal_validation,
				)

				continue
			} else {
				log.Print("encode_schedule: [passed] horizontal validation")
			}

			// save genetic algorithm's generated university schedule

			if err := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(fittest_uni_sched, semester_to_encode); err != nil {
				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status: RouteGlobals.SchedGenStatusInternalError,
						Message: fmt.Sprintf(
							"error saving schedule after %s, caused by : %s",
							time.Since(start),
							err.Error(),
						),
					},
				)

				log.Print("encode_schedule: [save-failed] error unable to save the genetic algorithm's generated schedule, caused by :", err.Error())
				continue queue_pop_loop
			}

			log.Print("encode_schedule: genetic algorithm's generated schedule saved successfully")

			// cache genetic algorithm's generated university schedule

			if err := RouteGlobals.SetCachedUniversitySchedule(semester_to_encode, fittest_uni_sched); err != nil {
				RouteGlobals.SetDeptSchedGenResult(
					RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
					RouteGlobals.SchedGenResult{
						Status: RouteGlobals.SchedGenStatusInternalError,
						Message: fmt.Sprintf(
							"error caching schedule after %s, caused by %s",
							time.Since(start),
							err.Error(),
						),
					},
				)

				log.Print("encode_schedule: [cache-failed] unable to change the genetic algorithm's generated schedule:", err.Error())
			}

			log.Print("encode_schedule: genetic algorithm's generated schedule cached successfully")

			// check if other department schedules are broken during the process

			is_other_dept_valid_final := make(map[uint16]bool)

			for other_dept_id := range dept_id_to_department {

				if department_to_encode[other_dept_id] {
					continue
				}

				other_dept_to_validate := make(map[uint16]bool)
				other_dept_to_validate[other_dept_id] = true

				errs_hv := GeneticAlgorithm.HorizontalValidation(
					fittest_uni_sched, curriculums, other_dept_to_validate, semester_to_encode,
				)

				is_other_dept_valid_final[other_dept_id] = len(errs_hv) == 0

				log.Printf(
					"encode_schedule: [other-sched-final] %s %s no errors: %t",
					dept_id_to_department[other_dept_id].Code,
					Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					is_other_dept_valid_final[other_dept_id],
				)
			}

			for other_dept_id := range dept_id_to_department {

				if department_to_encode[other_dept_id] {
					continue
				}

				is_initial_valid := is_other_dept_valid_initial[other_dept_id]
				is_final_valid := is_other_dept_valid_final[other_dept_id]

				if is_initial_valid && !is_final_valid {
					log.Printf(
						"encode_schedule: [broke-others] genetic algorithm accidentally broke the schedules of %s %s",
						dept_id_to_department[other_dept_id].Code, Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
					)

					RouteGlobals.SetDeptSchedGenResult(
						RouteGlobals.DeptSchedGenKey{DepartmentID: other_dept_id, Semester: semester_to_encode},
						RouteGlobals.SchedGenResult{
							Status: RouteGlobals.SchedGenStatusInternalError,
							Message: fmt.Sprintf(
								"your schedule might have been affected when %s finished generating schedules for %s, please try to validate your schedule by pressing the orange 'VALIDATE SCHEDULES' button",
								dept_id_to_department[other_dept_id].Code, Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
							),
						},
					)
				}
			}

			// specific department schedule generation done

			break
		}

		// set result of schedule generation for the current department to success

		RouteGlobals.SetDeptSchedGenResult(
			RouteGlobals.DeptSchedGenKey{DepartmentID: department_id, Semester: semester_to_encode},
			RouteGlobals.SchedGenResult{
				Status:                       RouteGlobals.SchedGenStatusSuccess,
				Message:                      fmt.Sprintf("schedule generation done after %s", time.Since(start)),
				FitnessProgressionDepartment: fitness_progression_department,
				FitnessProgressionUniversity: fitness_progression_university,
			},
		)

		log.Printf(
			"encode_schedule: [success] genetic algorithm schedule generation success for %s %s after %d tries, %s",
			dept_id_to_department[department_id].Code,
			Curriculum.SEMESTER_INDEX_NAME[semester_to_encode],
			retry+1, time.Since(start),
		)
	}

	log.Println("encode_schedule: [function ended]")
}

func has_department_to_encode(department_to_encode map[uint16]bool) bool {
	has_department_to_encode_result := false

	if department_to_encode == nil {
		return has_department_to_encode_result
	}

	if len(department_to_encode) == 0 {
		return has_department_to_encode_result
	}

	for _, v := range department_to_encode {
		has_department_to_encode_result = has_department_to_encode_result || v
	}

	return has_department_to_encode_result
}
