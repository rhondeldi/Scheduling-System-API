package StorageSchedule

import (
	"context"
	"log"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongodbReader struct {
	Mongo *MongoDB
}

func (s *MongodbReader) LoadSchedules(semester int) (Schedule.UniTimeTables, error) {

	if s.Mongo.Schedules == nil {
		s.Mongo.Schedules = s.Mongo.Client.Database("gass").Collection("schedules")
	}

	schedule_collection := s.Mongo.Schedules

	opt := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	mongodb_schedule := &MongodbSchedule{}

	err := schedule_collection.FindOne(
		context.TODO(),

		bson.D{{
			Key:   "Semester",
			Value: semester,
		}},

		opt,
	).Decode(mongodb_schedule)

	if err != nil {
		log.Println("LoadSchedules FindOne error:", err)
		return nil, err
	}

	deserialize_data := Schedule.DeserializeUniversitySchedule(mongodb_schedule.Schedule)

	return deserialize_data, nil
}
