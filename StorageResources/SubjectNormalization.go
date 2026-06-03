package StorageResources

import "github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"

func normalizeSubject(subject *Curriculum.Subject) {
	if subject == nil {
		return
	}
	subject.NormalizeAsyncConfig()
}

func normalizeSubjects(subjects []Curriculum.Subject) {
	for i := range subjects {
		normalizeSubject(&subjects[i])
	}
}

func normalizeCurriculumSubjects(curriculum *Curriculum.Curriculum) {
	if curriculum == nil {
		return
	}

	for yearLevelIdx := range curriculum.YearLevels {
		for semesterIdx := range curriculum.YearLevels[yearLevelIdx].Semesters {
			normalizeSubjects(curriculum.YearLevels[yearLevelIdx].Semesters[semesterIdx].Subjects)
		}
	}
}

func normalizeCurriculums(curriculums []Curriculum.Curriculum) {
	for i := range curriculums {
		normalizeCurriculumSubjects(&curriculums[i])
	}
}
