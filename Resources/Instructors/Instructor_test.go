package Instructors

import (
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

func TestInstructorEmploymentValidationAndCap(t *testing.T) {
	cases := []struct {
		name              string
		in                Instructor
		wantErr           bool
		wantEmploymentTyp string
		wantEffectiveMax  uint8
	}{
		{
			name:              "empty employment type defaults to regular at regular cap",
			in:                Instructor{FirstName: "Ada", LastName: "Lovelace"},
			wantErr:           false,
			wantEmploymentTyp: EMPLOYMENT_TYPE_REGULAR,
			wantEffectiveMax:  Const.REGULAR_INSTRUCTOR_MAX_UNITS,
		},
		{
			name:              "regular instructor is forced to the regular cap even if MaxUnits set lower",
			in:                Instructor{FirstName: "Alan", LastName: "Turing", EmploymentType: "regular", MaxUnits: 10},
			wantErr:           false,
			wantEmploymentTyp: EMPLOYMENT_TYPE_REGULAR,
			wantEffectiveMax:  Const.REGULAR_INSTRUCTOR_MAX_UNITS,
		},
		{
			name:              "part-time with explicit cap below regular is accepted",
			in:                Instructor{FirstName: "Grace", LastName: "Hopper", EmploymentType: "Part-Time", MaxUnits: 12},
			wantErr:           false,
			wantEmploymentTyp: EMPLOYMENT_TYPE_PART_TIME,
			wantEffectiveMax:  12,
		},
		{
			name:              "part-time with zero cap defaults to one below regular",
			in:                Instructor{FirstName: "Edsger", LastName: "Dijkstra", EmploymentType: "part-time"},
			wantErr:           false,
			wantEmploymentTyp: EMPLOYMENT_TYPE_PART_TIME,
			wantEffectiveMax:  Const.REGULAR_INSTRUCTOR_MAX_UNITS - 1,
		},
		{
			name:    "part-time with cap at or above regular is rejected",
			in:      Instructor{FirstName: "Ken", LastName: "Thompson", EmploymentType: "part-time", MaxUnits: Const.REGULAR_INSTRUCTOR_MAX_UNITS},
			wantErr: true,
		},
		{
			name:    "missing last name is rejected",
			in:      Instructor{FirstName: "NoLast"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			instructor := tc.in
			err := instructor.Validate()

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected validation error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}

			if instructor.EmploymentType != tc.wantEmploymentTyp {
				t.Errorf("employment type = %q, want %q", instructor.EmploymentType, tc.wantEmploymentTyp)
			}

			if got := instructor.EffectiveMaxUnits(); got != tc.wantEffectiveMax {
				t.Errorf("EffectiveMaxUnits() = %d, want %d", got, tc.wantEffectiveMax)
			}
		})
	}
}
