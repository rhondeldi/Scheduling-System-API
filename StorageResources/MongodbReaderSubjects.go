package StorageResources

import (
	"context"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbReader) ReadAllSubjects() ([]Curriculum.Subject, error) {

	if s.Mongo.Subjects == nil {
		s.Mongo.Subjects = s.Mongo.Client.Database("gass").Collection("subjects")
	}

	subject_collection := s.Mongo.Subjects

	find_opts := options.Find().SetSort(bson.D{{Key: "ID", Value: 1}}).SetProjection(bson.D{{Key: "_id", Value: 0}})

	cursor, err := subject_collection.Find(context.TODO(), bson.D{{}}, find_opts)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		cursor.Close(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	subjects := make([]Curriculum.Subject, 0)

	for cursor.Next(ctx) {
		subject := &Curriculum.Subject{}

		err := cursor.Decode(subject)

		if err != nil {
			log.Println("ReadAllSubjects: cursor.Next() error:")
			panic(err)
		}

		subjects = append(subjects, *subject)
	}

	return subjects, nil
}

func (s *MongodbReader) ReadSubject(subject_id uint16) (*Curriculum.Subject, error) {

	if s.Mongo.Subjects == nil {
		s.Mongo.Subjects = s.Mongo.Client.Database("gass").Collection("subjects")
	}

	subject_collection := s.Mongo.Subjects

	subject := &Curriculum.Subject{}

	filter := bson.D{{Key: "ID", Value: subject_id}}
	opts := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	err := subject_collection.FindOne(context.TODO(), filter, opts).Decode(subject)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	return subject, nil
}
