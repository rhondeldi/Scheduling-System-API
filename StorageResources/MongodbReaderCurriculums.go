package StorageResources

import (
	"context"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbReader) ReadAllCurriculum() ([]Curriculum.Curriculum, error) {

	if s.Mongo.Curriculums == nil {
		s.Mongo.Curriculums = s.Mongo.Client.Database("gass").Collection("curriculums")
	}

	curriculum_collection := s.Mongo.Curriculums

	find_opts := options.Find().SetSort(bson.D{{Key: "CurriculumID", Value: 1}}).SetProjection(bson.D{{Key: "_id", Value: 0}})

	cursor, err := curriculum_collection.Find(context.TODO(), bson.D{{}}, find_opts)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		cursor.Close(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	curriculums := make([]Curriculum.Curriculum, 0)

	for cursor.Next(ctx) {
		curriculum := &Curriculum.Curriculum{}

		err := cursor.Decode(curriculum)

		if err != nil {
			log.Println("ReadAllCurriculums: cursor.Next() error:")
			panic(err)
		}

		curriculums = append(curriculums, *curriculum)
	}

	return curriculums, nil
}

func (s *MongodbReader) ReadCurriculum(curriculum_id uint16) (*Curriculum.Curriculum, error) {

	if s.Mongo.Curriculums == nil {
		s.Mongo.Curriculums = s.Mongo.Client.Database("gass").Collection("curriculums")
	}

	curriculum_collection := s.Mongo.Curriculums

	curriculum := &Curriculum.Curriculum{}

	filter := bson.D{{Key: "CurriculumID", Value: curriculum_id}}
	opts := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	err := curriculum_collection.FindOne(context.TODO(), filter, opts).Decode(curriculum)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	return curriculum, nil
}
