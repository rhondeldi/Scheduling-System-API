package StorageResources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbWriter) CreateInstructor(arg_instructor Instructors.Instructor) error {

	new_instructor := Instructors.InstructorWithTimeString{
		InstructorID:  arg_instructor.InstructorID,
		DepartmentID:  arg_instructor.DepartmentID,
		FirstName:     arg_instructor.FirstName,
		MiddleInitial: arg_instructor.MiddleInitial,
		LastName:      arg_instructor.LastName,
		DesignatedSubjectIDs: arg_instructor.DesignatedSubjectIDs,
	}

	new_instructor.Time = arg_instructor.Time.Stringify()

	if new_instructor.InstructorID != 0 {
		return errors.New("error CreateInstructor(): cannot create a new instructor with a non-zero InstructorID")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	///// get the last instructor /////

	if s.Mongo.Instructors == nil {
		s.Mongo.Instructors = s.Mongo.Client.Database("gass").Collection("instructors")
	}

	opt_find_last := options.Find().
		SetSort(bson.D{{Key: "InstructorID", Value: -1}}).
		SetLimit(1).
		SetProjection(bson.D{{Key: "_id", Value: 0}})

	instructor_collection := s.Mongo.Instructors
	cursor_last, err_find_last := instructor_collection.Find(context.TODO(), bson.D{{}}, opt_find_last)

	if err_find_last != nil {
		return fmt.Errorf("CreateInstructor Find() error: %s", err_find_last.Error())
	}

	defer func() {
		cursor_last.Close(context.Background())
	}()

	ctx_find_last, cancel_find_last := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel_find_last()

	last_instructor := &Instructors.InstructorWithTimeString{}

	for cursor_last.Next(ctx_find_last) {
		err_decode_last := cursor_last.Decode(last_instructor)

		if err_decode_last != nil {
			log.Println("CreateInstructor Decode() error:", cursor_last)
			return err_decode_last
		}
	}

	///// save new curriculm instructor /////

	save_new_instructor := new_instructor
	save_new_instructor.InstructorID = last_instructor.InstructorID + 1

	insert_one_result, err_insert_one := instructor_collection.InsertOne(context.TODO(), save_new_instructor)

	if err_insert_one != nil {
		log.Println("CreateInstructor: InsertOne() error:", err_insert_one)
		return err_insert_one
	}

	log.Println("CreateInstructor: InsertOne Result:", insert_one_result)

	return nil
}

func (s *MongodbWriter) UpdateInstructor(arg_updated_instructor Instructors.Instructor) error {

	updated_instructor := Instructors.InstructorWithTimeString{
		InstructorID:  arg_updated_instructor.InstructorID,
		DepartmentID:  arg_updated_instructor.DepartmentID,
		FirstName:     arg_updated_instructor.FirstName,
		MiddleInitial: arg_updated_instructor.MiddleInitial,
		LastName:      arg_updated_instructor.LastName,
		DesignatedSubjectIDs: arg_updated_instructor.DesignatedSubjectIDs,
	}

	updated_instructor.Time = arg_updated_instructor.Time.Stringify()

	if updated_instructor.InstructorID == 0 {
		return errors.New("error UpdateInstructor(): parameter argument missing invalid InstructorID")
	}

	if s.Mongo.Instructors == nil {
		s.Mongo.Instructors = s.Mongo.Client.Database("gass").Collection("instructors")
	}

	instructor_collection := s.Mongo.Instructors

	replace_result, err_replace_one := instructor_collection.ReplaceOne(
		context.TODO(),
		bson.D{{Key: "InstructorID", Value: updated_instructor.InstructorID}},
		updated_instructor,
	)

	log.Println("UpdateInstructor ReplaceOne result:", replace_result)

	return err_replace_one
}

func (s *MongodbWriter) DeleteInstructor(instructor_id uint16) error {

	if instructor_id == 0 {
		return errors.New("parameter argument missing invalid instructor ID")
	}

	if s.Mongo.Instructors == nil {
		s.Mongo.Instructors = s.Mongo.Client.Database("gass").Collection("instructors")
	}

	instructor_collection := s.Mongo.Instructors

	delete_result, err_delete_one := instructor_collection.DeleteOne(
		context.TODO(),
		bson.D{{Key: "InstructorID", Value: instructor_id}},
	)

	log.Println("UpdateInstructor DeleteOne result:", delete_result)

	return err_delete_one
}
