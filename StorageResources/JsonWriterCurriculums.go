package StorageResources

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) CreateCurriculum(new_curriculum Curriculum.Curriculum) (uint16, error) {

	if new_curriculum.CurriculumID != 0 {
		return 0, errors.New("error CreateCurriculum(): cannot create a new curriculum with a non-zero CurriculumID")
	}

	CurriculumsMutex.Lock()
	defer CurriculumsMutex.Unlock()

	all_curriculums, err_read := json_read_all_curriculums()

	if err_read != nil {
		return 0, fmt.Errorf("error CreateCurriculum(): %s", err_read.Error())
	}

	new_curriculum_id := uint16(1)

	if len(all_curriculums) > 0 {
		new_curriculum_id = all_curriculums[len(all_curriculums)-1].CurriculumID + 1
	}

	err_save := json_save_curriculum(
		Curriculum.Curriculum{
			CurriculumID:   new_curriculum_id,
			CurriculumName: new_curriculum.CurriculumName,
			CurriculumCode: new_curriculum.CurriculumCode,
			DepartmentID:   new_curriculum.DepartmentID,
			YearLevels:     new_curriculum.YearLevels,
		},
	)

	if err_save != nil {
		return 0, fmt.Errorf("error CreateCurriculum(): %s", err_save.Error())
	}

	return new_curriculum_id, nil
}

func (s *JsonWriter) UpdateCurriculum(updated_curriculum Curriculum.Curriculum) error {

	if updated_curriculum.CurriculumID == 0 {
		return errors.New("error UpdateCurriculum(): parameter argument missing invalid CurriculumID")
	}

	CurriculumsMutex.Lock()
	defer CurriculumsMutex.Unlock()

	all_curriculums, err_read := json_read_all_curriculums()

	if err_read != nil {
		return fmt.Errorf("error UpdateCurriculum(): %s", err_read.Error())
	}

	has_id := false

	for _, curriculum := range all_curriculums {
		if curriculum.CurriculumID == updated_curriculum.CurriculumID {
			has_id = true
			break
		}
	}

	if !has_id {
		return fmt.Errorf(
			"error UpdateCurriculum(): curriculum %d %s - %s does not exist in the json file",
			updated_curriculum.CurriculumID, updated_curriculum.CurriculumCode, updated_curriculum.CurriculumName,
		)
	}

	err_edit_curriculums := json_edit_curriculum(updated_curriculum)

	if err_edit_curriculums != nil {
		return fmt.Errorf("error UpdateCurriculum(): %s", err_edit_curriculums.Error())
	}

	return nil
}

func (s *JsonWriter) DeleteCurriculum(curriculum_id uint16) error {

	if curriculum_id == 0 {
		return errors.New("parameter argument missing invalid curriculum ID")
	}

	CurriculumsMutex.Lock()
	defer CurriculumsMutex.Unlock()

	all_curriculums, err_read := json_read_all_curriculums()

	if err_read != nil {
		return err_read
	}

	other_curriculums := make([]Curriculum.Curriculum, 0)

	has_id := false

	for _, curriculum := range all_curriculums {
		if curriculum.CurriculumID == curriculum_id {
			has_id = true
		} else {
			other_curriculums = append(other_curriculums, curriculum)
		}
	}

	if !has_id {
		return errors.New("curriculum to delete does not exist in the json file")
	}

	err_save_curriculums := json_save_all_curriculums(other_curriculums)

	if err_save_curriculums != nil {
		return err_save_curriculums
	}

	return nil
}

// no op if curriculum slice is empty
func json_save_all_curriculums(curriculums []Curriculum.Curriculum) error {
	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	curriculums_json_file := path.Join(project_root, "scheduling-system-temporary-data", "curriculums.json")

	sort.Slice(curriculums, func(i, j int) bool {
		return curriculums[i].CurriculumID < curriculums[j].CurriculumID
	})

	curriculums_byte_data, err := json.MarshalIndent(curriculums, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(curriculums_json_file, curriculums_byte_data, 0644); err != nil {
		return err
	}

	return nil
}

func json_save_curriculum(new_curriculum Curriculum.Curriculum) error {

	// read all curriculums

	all_curriculums, err_read_all_curriculums := json_read_all_curriculums()

	if err_read_all_curriculums != nil {
		return err_read_all_curriculums
	}

	// check if curriculum to save already exist

	for _, curriculum := range all_curriculums {
		if curriculum.CurriculumID == new_curriculum.CurriculumID {

			// return error if it exists

			return fmt.Errorf(
				"the CurriculumID '%d' already exist: '%s - %s'",
				curriculum.CurriculumID, curriculum.CurriculumCode, curriculum.CurriculumName,
			)
		}
	}

	// append the new curriculum if it does not exist

	all_curriculums = append(all_curriculums, new_curriculum)

	// save all the curriculums with the new added curriculum

	err_save_all_curriculums := json_save_all_curriculums(all_curriculums)

	if err_save_all_curriculums != nil {
		return err_read_all_curriculums
	}

	return nil
}

func json_edit_curriculum(edited_curriculum Curriculum.Curriculum) error {

	// read all curriculums

	all_curriculums, err_read_all_curriculums := json_read_all_curriculums()

	if err_read_all_curriculums != nil {
		return err_read_all_curriculums
	}

	// check if curriculum to edit exists

	has_curriculum := false

	for curriculum_idx, curriculum := range all_curriculums {
		if curriculum.CurriculumID == edited_curriculum.CurriculumID {
			has_curriculum = true

			all_curriculums[curriculum_idx] = edited_curriculum // apply edit if found

			break
		}
	}

	// return error if it does not exist

	if !has_curriculum {
		return fmt.Errorf(
			"the CurriculumID '%d' does not exist", edited_curriculum.CurriculumID,
		)
	}

	// save all curriculums with the edited curriculum

	err_save_all_curriculums := json_save_all_curriculums(all_curriculums)

	if err_save_all_curriculums != nil {
		return err_read_all_curriculums
	}

	return nil
}
