package StorageResources

import (
	"context"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbReader) ReadAllInstructors() ([]Instructors.Instructor, error) {

	instructors_with_time_string, err := s.ReadAllInstructorsWithTimeString()

	if err != nil {
		return nil, err
	}

	instructors := make([]Instructors.Instructor, 0)

	for _, instructor_with_time_str := range instructors_with_time_string {
		instructor := Instructors.Instructor{
			InstructorID:  instructor_with_time_str.InstructorID,
			DepartmentID:  instructor_with_time_str.DepartmentID,
			FirstName:     instructor_with_time_str.FirstName,
			MiddleInitial: instructor_with_time_str.MiddleInitial,
			LastName:      instructor_with_time_str.LastName,
			EmploymentType: instructor_with_time_str.EmploymentType,
			MaxUnits:       instructor_with_time_str.MaxUnits,
			DesignatedSubjectIDs: instructor_with_time_str.DesignatedSubjectIDs,
		}

		instructor.Time.StringParse(instructor_with_time_str.Time)

		instructors = append(instructors, instructor)
	}

	return instructors, nil
}

func (s *MongodbReader) ReadInstructor(instructor_id uint16) (*Instructors.Instructor, error) {

	if s.Mongo.Instructors == nil {
		s.Mongo.Instructors = s.Mongo.Client.Database("gass").Collection("instructors")
	}

	instructor_collection := s.Mongo.Instructors
	instructor_with_time_str := &Instructors.InstructorWithTimeString{}

	filter := bson.D{{Key: "InstructorID", Value: instructor_id}}
	opts := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	err := instructor_collection.FindOne(context.TODO(), filter, opts).Decode(instructor_with_time_str)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	instructor := &Instructors.Instructor{
		InstructorID:  instructor_with_time_str.InstructorID,
		DepartmentID:  instructor_with_time_str.DepartmentID,
		FirstName:     instructor_with_time_str.FirstName,
		MiddleInitial: instructor_with_time_str.MiddleInitial,
		LastName:      instructor_with_time_str.LastName,
		EmploymentType: instructor_with_time_str.EmploymentType,
		MaxUnits:       instructor_with_time_str.MaxUnits,
		DesignatedSubjectIDs: instructor_with_time_str.DesignatedSubjectIDs,
	}

	instructor.Time.StringParse(instructor_with_time_str.Time)

	return instructor, nil
}

func (s *MongodbReader) ReadDepartmentInstructors(department_id int) ([]Instructors.Instructor, error) {

	instructors_with_time_string, err := s.ReadAllInstructorsWithTimeString()

	if err != nil {
		return nil, err
	}

	department_instructors := make([]Instructors.Instructor, 0)

	for _, instructor_with_time_str := range instructors_with_time_string {
		if instructor_with_time_str.DepartmentID != uint16(department_id) {
			continue
		}

		instructor := Instructors.Instructor{
			InstructorID:  instructor_with_time_str.InstructorID,
			DepartmentID:  instructor_with_time_str.DepartmentID,
			FirstName:     instructor_with_time_str.FirstName,
			MiddleInitial: instructor_with_time_str.MiddleInitial,
			LastName:      instructor_with_time_str.LastName,
			EmploymentType: instructor_with_time_str.EmploymentType,
			MaxUnits:       instructor_with_time_str.MaxUnits,
			DesignatedSubjectIDs: instructor_with_time_str.DesignatedSubjectIDs,
		}

		instructor.Time.StringParse(instructor_with_time_str.Time)

		department_instructors = append(department_instructors, instructor)
	}

	return department_instructors, nil
}

func (s *MongodbReader) ReadAllInstructorsWithTimeString() ([]Instructors.InstructorWithTimeString, error) {
	if s.Mongo.Instructors == nil {
		s.Mongo.Instructors = s.Mongo.Client.Database("gass").Collection("instructors")
	}

	instructor_collection := s.Mongo.Instructors

	find_opts := options.Find().SetSort(bson.D{{Key: "InstructorID", Value: 1}}).SetProjection(bson.D{{Key: "_id", Value: 0}})

	cursor, err := instructor_collection.Find(context.TODO(), bson.D{{}}, find_opts)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		cursor.Close(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	instructors := make([]Instructors.InstructorWithTimeString, 0)

	for cursor.Next(ctx) {
		instructor := &Instructors.InstructorWithTimeString{}

		err := cursor.Decode(instructor)

		if err != nil {
			log.Println("ReadAllInstructorsWithTimeString: cursor.Next() error:")
			panic(err)
		}

		instructors = append(instructors, *instructor)
	}

	return instructors, nil
}
