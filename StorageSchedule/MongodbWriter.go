package StorageSchedule

import (
	"context"
	"fmt"
	"log"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbWriter struct {
	Mongo *MongoDB
}

type MongodbSchedule struct {
	Semester int    `bson:"Semester"`
	Schedule []byte `bson:"Schedule"`
}

func (s *MongodbWriter) SaveSchedules(university_schedule Schedule.UniTimeTables, semester int) error {

	if s.Mongo.Schedules == nil {
		s.Mongo.Schedules = s.Mongo.Client.Database("gass").Collection("schedules")
	}

	schedule_collection := s.Mongo.Schedules

	opt := options.Replace().SetUpsert(true)

	result, err := schedule_collection.ReplaceOne(
		context.TODO(),

		bson.D{{
			Key:   "Semester",
			Value: semester,
		}},

		&MongodbSchedule{
			Semester: semester,
			Schedule: Schedule.SerializeUniversitySchedule(
				university_schedule,
			),
		},

		opt,
	)

	log.Println("SaveSchedules ReplaceOne result:", result)

	return err
}

func (s *MongodbWriter) DeleteSchedules(semester int) error {

	if s.Mongo.Schedules == nil {
		s.Mongo.Schedules = s.Mongo.Client.Database("gass").Collection("schedules")
	}

	schedule_collection := s.Mongo.Schedules

	result, err := schedule_collection.DeleteOne(
		context.TODO(),

		bson.D{{
			Key:   "Semester",
			Value: semester,
		}},
	)

	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("no document found for semester %d", semester)
	}

	return nil
}
