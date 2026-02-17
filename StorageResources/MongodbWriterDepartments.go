package StorageResources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbWriter) CreateDepartment(new_department Departments.Department) error {

	if new_department.DepartmentID != 0 {
		return errors.New("error CreateDepartment(): cannot create a new department with a non-zero DepartmentID")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	///// get the last department /////

	if s.Mongo.Departments == nil {
		s.Mongo.Departments = s.Mongo.Client.Database("gass").Collection("departments")
	}

	opt_find_last := options.Find().
		SetSort(bson.D{{Key: "DepartmentID", Value: -1}}).
		SetLimit(1).
		SetProjection(bson.D{{Key: "_id", Value: 0}})

	department_collection := s.Mongo.Departments
	cursor_last, err_find_last := department_collection.Find(context.TODO(), bson.D{{}}, opt_find_last)

	if err_find_last != nil {
		return fmt.Errorf("CreateDepartment Find() error: %s", err_find_last.Error())
	}

	defer func() {
		cursor_last.Close(context.Background())
	}()

	ctx_find_last, cancel_find_last := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel_find_last()

	last_department := &Departments.Department{}

	for cursor_last.Next(ctx_find_last) {
		err_decode_last := cursor_last.Decode(last_department)

		if err_decode_last != nil {
			log.Println("CreateDepartment Decode() error:", cursor_last)
			return err_decode_last
		}
	}

	///// save new curriculm department /////

	save_new_department := new_department
	save_new_department.DepartmentID = last_department.DepartmentID + 1

	insert_one_result, err_insert_one := department_collection.InsertOne(context.TODO(), save_new_department)

	if err_insert_one != nil {
		log.Println("CreateDepartment: InsertOne() error:", err_insert_one)
		return err_insert_one
	}

	log.Println("CreateDepartment: InsertOne Result:", insert_one_result)

	return nil
}

func (s *MongodbWriter) UpdateDepartment(updated_department Departments.Department) error {

	if updated_department.DepartmentID == 0 {
		return errors.New("error UpdateDepartment(): parameter argument missing invalid DepartmentID")
	}

	if s.Mongo.Departments == nil {
		s.Mongo.Departments = s.Mongo.Client.Database("gass").Collection("departments")
	}

	department_collection := s.Mongo.Departments

	replace_result, err_replace_one := department_collection.ReplaceOne(
		context.TODO(),
		bson.D{{Key: "DepartmentID", Value: updated_department.DepartmentID}},
		updated_department,
	)

	log.Println("UpdateDepartment ReplaceOne result:", replace_result)

	return err_replace_one
}

func (s *MongodbWriter) DeleteDepartment(department_id uint16) error {

	if department_id == 0 {
		return errors.New("parameter argument missing invalid department ID")
	}

	if s.Mongo.Departments == nil {
		s.Mongo.Departments = s.Mongo.Client.Database("gass").Collection("departments")
	}

	department_collection := s.Mongo.Departments

	delete_result, err_delete_one := department_collection.DeleteOne(
		context.TODO(),
		bson.D{{Key: "DepartmentID", Value: department_id}},
	)

	log.Println("UpdateDepartment DeleteOne result:", delete_result)

	return err_delete_one
}
