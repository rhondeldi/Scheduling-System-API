package StorageResources

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"sort"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func (s *JsonWriter) CreateSubject(new_subject Curriculum.Subject) error {

	if new_subject.ID != 0 {
		return errors.New("cannot create a new subject with a non zero subject ID because that would overwrite a subject item")
	}

	new_subject.NormalizeAsyncConfig()

	SubjectMutex.Lock()
	defer SubjectMutex.Unlock()

	all_subject, err_read := json_read_all_subjects()

	if err_read != nil {
		return err_read
	}

	new_id_num := uint16(1)

	if len(all_subject) > 0 {
		new_id_num = all_subject[len(all_subject)-1].ID + 1
	}

	all_subject = append(all_subject, Curriculum.Subject{
		ID:                    new_id_num,
		Code:                  new_subject.Code,
		Name:                  new_subject.Name,
		LecHours:              new_subject.LecHours,
		LabHours:              new_subject.LabHours,
		SubjectType:           new_subject.SubjectType,
		AsynchronousHours:     new_subject.AsynchronousHours,
		SynchronousHours:      new_subject.SynchronousHours,
		SaturdayOnly:          new_subject.SaturdayOnly,
		BitFlags:              new_subject.BitFlags,
		DesignatedInstructors: new_subject.DesignatedInstructors,
	})

	err_save_subjects := json_save_all_subjects(all_subject)

	if err_save_subjects != nil {
		return err_save_subjects
	}

	return nil
}

func (s *JsonWriter) UpdateSubject(subject_to_update Curriculum.Subject) error {

	if subject_to_update.ID == 0 {
		return errors.New("parameter argument missing invalid subject ID")
	}

	subject_to_update.NormalizeAsyncConfig()

	SubjectMutex.Lock()
	defer SubjectMutex.Unlock()

	all_subjects, err_read := json_read_all_subjects()

	if err_read != nil {
		return err_read
	}

	all_curriculums, err_read_all_curriculums := json_read_all_curriculums()

	if err_read_all_curriculums != nil {
		return err_read_all_curriculums
	}

	has_id := false
	to_update_idx := -1

	for idx, subject := range all_subjects {
		if subject.ID == subject_to_update.ID {
			has_id = true
			to_update_idx = idx
			break
		}
	}

	if !has_id {
		return errors.New("subject to update does not exist in the json file")
	}

	updated_subject := Curriculum.Subject{
		ID:                    subject_to_update.ID,
		Code:                  subject_to_update.Code,
		Name:                  subject_to_update.Name,
		LecHours:              subject_to_update.LecHours,
		LabHours:              subject_to_update.LabHours,
		SubjectType:           subject_to_update.SubjectType,
		AsynchronousHours:     subject_to_update.AsynchronousHours,
		SynchronousHours:      subject_to_update.SynchronousHours,
		SaturdayOnly:          subject_to_update.SaturdayOnly,
		BitFlags:              subject_to_update.BitFlags,
		DesignatedInstructors: subject_to_update.DesignatedInstructors,
	}

	all_subjects[to_update_idx] = updated_subject

	for curriculum_idx := range all_curriculums {
		for yrlvl_idx := range all_curriculums[curriculum_idx].YearLevels {
			for semester_idx := range all_curriculums[curriculum_idx].YearLevels[yrlvl_idx].Semesters {
				for subject_idx := range all_curriculums[curriculum_idx].YearLevels[yrlvl_idx].Semesters[semester_idx].Subjects {
					curriculum_subject_id := all_curriculums[curriculum_idx].YearLevels[yrlvl_idx].Semesters[semester_idx].Subjects[subject_idx].ID
					if subject_to_update.ID == curriculum_subject_id {
						all_curriculums[curriculum_idx].YearLevels[yrlvl_idx].Semesters[semester_idx].Subjects[subject_idx] = updated_subject
					}
				}
			}
		}
	}

	err_save_subjects := json_save_all_subjects(all_subjects)

	if err_save_subjects != nil {
		return err_save_subjects
	}

	err_save_curriculums := json_save_all_curriculums(all_curriculums)

	if err_save_curriculums != nil {
		return err_save_curriculums
	}

	return nil
}

func (s *JsonWriter) DeleteSubject(subject_id uint16) error {
	if subject_id == 0 {
		return errors.New("parameter argument missing invalid subject ID")
	}

	SubjectMutex.Lock()
	defer SubjectMutex.Unlock()

	all_subjects, err_read := json_read_all_subjects()

	if err_read != nil {
		return err_read
	}

	subjects_deleted := make([]Curriculum.Subject, 0)

	has_id := false

	for _, subject := range all_subjects {
		if subject.ID == subject_id {
			has_id = true
		} else {
			subjects_deleted = append(subjects_deleted, subject)
		}
	}

	if !has_id {
		return errors.New("subject to delete does not exist in the json file")
	}

	err_save_subjects := json_save_all_subjects(subjects_deleted)

	if err_save_subjects != nil {
		return err_save_subjects
	}

	return nil
}

func json_save_all_subjects(subjects []Curriculum.Subject) error {

	project_root, err_project_root := Utils.FindProjectRoot()

	if err_project_root != nil {
		return err_project_root
	}

	subjects_json_file := path.Join(project_root, "scheduling-system-temporary-data", "all-subjects.json")

	sort.Slice(subjects, func(i, j int) bool {
		return subjects[i].ID < subjects[j].ID
	})

	subjects_byte_data, err := json.MarshalIndent(subjects, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(subjects_json_file, subjects_byte_data, 0644); err != nil {
		return err
	}

	return nil
}
