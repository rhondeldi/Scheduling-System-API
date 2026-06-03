package StorageResources

import (
	"context"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbReader) ReadAsyncScheduleRecords(department_id uint16, semester int) ([]Schedule.AsyncScheduleRecord, error) {
	if s.Mongo.AsyncScheduleRecords == nil {
		s.Mongo.AsyncScheduleRecords = s.Mongo.Client.Database("gass").Collection("async_schedule_records")
	}

	collection := s.Mongo.AsyncScheduleRecords
	filter := bson.D{
		{Key: "DepartmentID", Value: department_id},
		{Key: "Semester", Value: semester},
	}
	findOpts := options.Find().
		SetProjection(bson.D{{Key: "_id", Value: 0}}).
		SetSort(bson.D{
			{Key: "InstructorID", Value: 1},
			{Key: "SubjectID", Value: 1},
			{Key: "SectionUSI", Value: 1},
		})

	cursor, err := collection.Find(context.TODO(), filter, findOpts)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer func() {
		cursor.Close(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	records := make([]Schedule.AsyncScheduleRecord, 0)
	for cursor.Next(ctx) {
		record := &Schedule.AsyncScheduleRecord{}
		if err := cursor.Decode(record); err != nil {
			log.Println("ReadAsyncScheduleRecords: cursor decode error")
			return nil, err
		}
		records = append(records, *record)
	}

	return records, nil
}
