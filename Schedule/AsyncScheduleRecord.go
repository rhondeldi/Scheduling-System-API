package Schedule

// AsyncScheduleRecord stores asynchronous (non-room) teaching load entries.
// These records are generated after GA completes and are kept separate from
// weekly grid slots.
type AsyncScheduleRecord struct {
	SectionID string `json:"SectionID" bson:"SectionID"`

	// USI is the section's index in UniTimeTables for the selected semester.
	SectionUSI int `json:"SectionUSI" bson:"SectionUSI"`

	CurriculumID uint16 `json:"CurriculumID" bson:"CurriculumID"`
	DepartmentID uint16 `json:"DepartmentID" bson:"DepartmentID"`
	YearLevelIdx int    `json:"YearLevelIdx" bson:"YearLevelIdx"`
	SectionIdx   int    `json:"SectionIdx" bson:"SectionIdx"`
	Semester     int    `json:"Semester" bson:"Semester"`
	Year         int    `json:"Year" bson:"Year"`

	SubjectID    uint16  `json:"SubjectID" bson:"SubjectID"`
	InstructorID uint16  `json:"InstructorID" bson:"InstructorID"`
	AsyncHours   float64 `json:"AsyncHours" bson:"AsyncHours"`

	// CourseSection is the section label in the standard "<CurriculumCode>-<Year><SectionLetter>" form,
	// e.g. "BSCS-1D" — matching how sync subjects are labeled in the timetable.
	CourseSection string `json:"CourseSection" bson:"CourseSection"`

	DisplayLabel string `json:"DisplayLabel" bson:"DisplayLabel"`
}
