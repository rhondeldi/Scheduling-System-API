package StorageResources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbWriter) CreateCurriculum(new_curriculum Curriculum.Curriculum) (uint16, error) {

	if new_curriculum.CurriculumID != 0 {
		return 0, errors.New("error CreateCurriculum(): cannot create a new curriculum with a non-zero CurriculumID")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	///// get the last curriculum /////

	if s.Mongo.Curriculums == nil {
		s.Mongo.Curriculums = s.Mongo.Client.Database("gass").Collection("curriculums")
	}

	opt_find_last := options.Find().
		SetSort(bson.D{{Key: "CurriculumID", Value: -1}}).
		SetLimit(1).
		SetProjection(bson.D{{Key: "_id", Value: 0}})

	curriculum_collection := s.Mongo.Curriculums
	cursor_last, err_find_last := curriculum_collection.Find(context.TODO(), bson.D{{}}, opt_find_last)

	if err_find_last != nil {
		return 0, fmt.Errorf("CreateCurriculum Find() error: %s", err_find_last.Error())
	}

	defer func() {
		cursor_last.Close(context.Background())
	}()

	ctx_find_last, cancel_find_last := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel_find_last()

	last_curriculum := &Curriculum.Curriculum{}

	for cursor_last.Next(ctx_find_last) {
		err_decode_last := cursor_last.Decode(last_curriculum)

		if err_decode_last != nil {
			log.Println("CreateCurriculum Decode() error:", cursor_last)
			return 0, err_decode_last
		}
	}

	///// save new curriculm curriculum /////

	new_curriculum_id := last_curriculum.CurriculumID + 1

	save_new_curriculum := &Curriculum.Curriculum{
		CurriculumID:   new_curriculum_id,
		CurriculumName: new_curriculum.CurriculumName,
		CurriculumCode: new_curriculum.CurriculumCode,
		DepartmentID:   new_curriculum.DepartmentID,
		YearLevels:     new_curriculum.YearLevels,
	}

	insert_one_result, err_insert_one := curriculum_collection.InsertOne(context.TODO(), save_new_curriculum)

	if err_insert_one != nil {
		log.Println("CreateCurriculum: InsertOne() error:", err_insert_one)
		return 0, err_insert_one
	}

	log.Println("CreateCurriculum: InsertOne Result:", insert_one_result)

	return new_curriculum_id, nil
}

func (s *MongodbWriter) UpdateCurriculum(updated_curriculum Curriculum.Curriculum) error {

	if updated_curriculum.CurriculumID == 0 {
		return errors.New("error UpdateCurriculum(): parameter argument missing invalid CurriculumID")
	}

	if s.Mongo.Curriculums == nil {
		s.Mongo.Curriculums = s.Mongo.Client.Database("gass").Collection("curriculums")
	}

	curriculum_collection := s.Mongo.Curriculums

	replace_result, err_replace_one := curriculum_collection.ReplaceOne(
		context.TODO(),
		bson.D{{Key: "CurriculumID", Value: updated_curriculum.CurriculumID}},
		updated_curriculum,
	)

	log.Println("UpdateCurriculum ReplaceOne result:", replace_result)

	return err_replace_one
}

func (s *MongodbWriter) DeleteCurriculum(curriculum_id uint16) error {

	if curriculum_id == 0 {
		return errors.New("parameter argument missing invalid curriculum ID")
	}

	if s.Mongo.Curriculums == nil {
		s.Mongo.Curriculums = s.Mongo.Client.Database("gass").Collection("curriculums")
	}

	curriculum_collection := s.Mongo.Curriculums

	delete_result, err_delete_one := curriculum_collection.DeleteOne(
		context.TODO(),
		bson.D{{Key: "CurriculumID", Value: curriculum_id}},
	)

	log.Println("UpdateCurriculum DeleteOne result:", delete_result)

	return err_delete_one
}
