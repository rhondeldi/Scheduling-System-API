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

func (s *MongodbWriter) CreateSubject(new_subject Curriculum.Subject) error {

	if new_subject.ID != 0 {
		return errors.New("error CreateSubject(): cannot create a new subject with a non-zero ID")
	}

	new_subject.NormalizeAsyncConfig()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	///// get the last subject /////

	if s.Mongo.Subjects == nil {
		s.Mongo.Subjects = s.Mongo.Client.Database("gass").Collection("subjects")
	}

	opt_find_last := options.Find().
		SetSort(bson.D{{Key: "ID", Value: -1}}).
		SetLimit(1).
		SetProjection(bson.D{{Key: "_id", Value: 0}})

	subject_collection := s.Mongo.Subjects
	cursor_last, err_find_last := subject_collection.Find(context.TODO(), bson.D{{}}, opt_find_last)

	if err_find_last != nil {
		return fmt.Errorf("CreateSubject Find() error: %s", err_find_last.Error())
	}

	defer func() {
		cursor_last.Close(context.Background())
	}()

	ctx_find_last, cancel_find_last := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel_find_last()

	last_subject := &Curriculum.Subject{}

	for cursor_last.Next(ctx_find_last) {
		err_decode_last := cursor_last.Decode(last_subject)

		if err_decode_last != nil {
			log.Println("CreateSubject Decode() error:", cursor_last)
			return err_decode_last
		}
	}

	///// save new curriculm subject /////

	save_new_subject := new_subject
	save_new_subject.ID = last_subject.ID + 1

	insert_one_result, err_insert_one := subject_collection.InsertOne(context.TODO(), save_new_subject)

	if err_insert_one != nil {
		log.Println("CreateSubject: InsertOne() error:", err_insert_one)
		return err_insert_one
	}

	log.Println("CreateSubject: InsertOne Result:", insert_one_result)

	return nil
}

func (s *MongodbWriter) UpdateSubject(updated_subject Curriculum.Subject) error {

	if updated_subject.ID == 0 {
		return errors.New("error UpdateSubject(): parameter argument missing invalid ID")
	}

	updated_subject.NormalizeAsyncConfig()

	if s.Mongo.Subjects == nil {
		s.Mongo.Subjects = s.Mongo.Client.Database("gass").Collection("subjects")
	}

	subject_collection := s.Mongo.Subjects

	replace_result, err_replace_one := subject_collection.ReplaceOne(
		context.TODO(),
		bson.D{{Key: "ID", Value: updated_subject.ID}},
		updated_subject,
	)

	log.Println("UpdateSubject ReplaceOne result:", replace_result)

	return err_replace_one
}

func (s *MongodbWriter) DeleteSubject(subject_id uint16) error {

	if subject_id == 0 {
		return errors.New("parameter argument missing invalid subject ID")
	}

	if s.Mongo.Subjects == nil {
		s.Mongo.Subjects = s.Mongo.Client.Database("gass").Collection("subjects")
	}

	subject_collection := s.Mongo.Subjects

	delete_result, err_delete_one := subject_collection.DeleteOne(
		context.TODO(),
		bson.D{{Key: "ID", Value: subject_id}},
	)

	log.Println("UpdateSubject DeleteOne result:", delete_result)

	return err_delete_one
}
