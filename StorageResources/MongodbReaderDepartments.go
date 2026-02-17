package StorageResources

import (
	"context"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbReader) ReadAllDepartments() ([]Departments.Department, error) {

	if s.Mongo.Departments == nil {
		s.Mongo.Departments = s.Mongo.Client.Database("gass").Collection("departments")
	}

	department_collection := s.Mongo.Departments

	find_opts := options.Find().SetSort(bson.D{{Key: "DepartmentID", Value: 1}}).SetProjection(bson.D{{Key: "_id", Value: 0}})

	cursor, err := department_collection.Find(context.TODO(), bson.D{{}}, find_opts)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		cursor.Close(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	departments := make([]Departments.Department, 0)

	for cursor.Next(ctx) {
		department := &Departments.Department{}

		err := cursor.Decode(department)

		if err != nil {
			log.Println("ReadAllDepartments: cursor.Next() error:")
			panic(err)
		}

		departments = append(departments, *department)
	}

	return departments, nil
}

func (s *MongodbReader) ReadDepartment(department_id uint16) (*Departments.Department, error) {

	if s.Mongo.Departments == nil {
		s.Mongo.Departments = s.Mongo.Client.Database("gass").Collection("departments")
	}

	department_collection := s.Mongo.Departments

	department := &Departments.Department{}

	filter := bson.D{{Key: "DepartmentID", Value: department_id}}
	opts := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	err := department_collection.FindOne(context.TODO(), filter, opts).Decode(department)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	return department, nil
}
