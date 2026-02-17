package Curriculum

type Curriculum struct {
	CurriculumID   uint16      `json:"CurriculumID" bson:"CurriculumID"`     // non negative & non-zero unique number even if for example we have Computer Science (OLD) and Computer Science (New)
	CurriculumName string      `json:"CurriculumName" bson:"CurriculumName"` // e.g. Computer Science, Information Technology
	CurriculumCode string      `json:"CurriculumCode" bson:"CurriculumCode"` // e.g. BSCS, BSIT
	DepartmentID   uint16      `json:"DepartmentID" bson:"DepartmentID"`
	YearLevels     []YearLevel `json:"YearLevels" bson:"YearLevels"`
}

type YearLevel struct {
	Name      string     `json:"Name" bson:"Name"`
	IsActive  bool       `json:"IsActive" bson:"IsActive"`
	Semesters []Semester `json:"Semesters" bson:"Semesters"`
}

type Semester struct {
	Name     string    `json:"Name" bson:"Name"`
	Sections int       `json:"Sections" bson:"Sections"`
	Subjects []Subject `json:"Subjects" bson:"Subjects"`
}

var SEMESTER_INDEX_NAME [6]string = [6]string{
	"1st semester",
	"2nd semester",
	"Mid-year",
	"4th semester",
	"5th semester",
	"6th semester",
}

const SUPPORTED_SEMESTERS int = 3

var SECTION [62]string = [62]string{
	// uppercase letters
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",

	// lowercase letters
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",

	// numbers
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
}

func GetTotalNumberOfSections(curriculums []Curriculum, selected_semester int) int {
	section_count := 0

	for _, curriculum := range curriculums {
		for _, year_level := range curriculum.YearLevels {

			if !year_level.IsActive {
				continue // skip inactive year levels
			}

			if selected_semester < 0 || selected_semester >= len(year_level.Semesters) {
				continue // skip invalid semester index
			}

			section_count += year_level.Semesters[selected_semester].Sections
		} // ------------- end of year_level loop -------------
	} // ------------- end of curriculum loop -------------

	return section_count
}

func (s *Curriculum) GetTotalSections() int {
	section_count := 0

	for _, year_level := range s.YearLevels {
		if !year_level.IsActive {
			continue
		}

		for _, semester := range year_level.Semesters {
			section_count += semester.Sections
		}
	}

	return section_count
}

func (s *Curriculum) GetTotalSectionsBySemester(semester_idx int) int {
	section_count := 0

	for _, year_level := range s.YearLevels {
		if !year_level.IsActive {
			continue
		}

		if semester_idx < 0 || semester_idx >= len(year_level.Semesters) {
			continue
		}

		section_count += year_level.Semesters[semester_idx].Sections
	}

	return section_count
}
