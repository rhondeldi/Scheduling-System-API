package StorageResources

import (
	"context"
	"log"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *MongodbWriter) ReplaceAsyncScheduleRecords(department_id uint16, semester int, records []Schedule.AsyncScheduleRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Mongo.AsyncScheduleRecords == nil {
		s.Mongo.AsyncScheduleRecords = s.Mongo.Client.Database("gass").Collection("async_schedule_records")
	}

	collection := s.Mongo.AsyncScheduleRecords
	filter := bson.D{
		{Key: "DepartmentID", Value: department_id},
		{Key: "Semester", Value: semester},
	}

	if _, err := collection.DeleteMany(context.TODO(), filter); err != nil {
		log.Println("ReplaceAsyncScheduleRecords: delete failed:", err)
		return err
	}

	if len(records) == 0 {
		return nil
	}

	docs := make([]any, 0, len(records))
	for _, record := range records {
		docs = append(docs, record)
	}

	if _, err := collection.InsertMany(context.TODO(), docs); err != nil {
		log.Println("ReplaceAsyncScheduleRecords: insert failed:", err)
		return err
	}

	return nil
}

func (s *MongodbWriter) DeleteAsyncScheduleRecords(department_id uint16, semester int) error {
	return s.ReplaceAsyncScheduleRecords(department_id, semester, nil)
}
