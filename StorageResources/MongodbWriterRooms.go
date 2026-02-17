package StorageResources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbWriter) CreateRoom(new_room Rooms.Room) error {

	if new_room.RoomID != 0 {
		return errors.New("error CreateRoom(): cannot create a new room with a non-zero RoomID")
	}

	if new_room.Capacity > uint16(Rooms.MAX_ROOM_CAPACITY) {
		return fmt.Errorf("MongoDB.CreateRoom room to add has a capacity of %d section(s), the maximum allowed is only 15", new_room.Capacity)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	///// get the last room /////

	if s.Mongo.Rooms == nil {
		s.Mongo.Rooms = s.Mongo.Client.Database("gass").Collection("rooms")
	}

	opt_find_last := options.Find().
		SetSort(bson.D{{Key: "RoomID", Value: -1}}).
		SetLimit(1).
		SetProjection(bson.D{{Key: "_id", Value: 0}})

	room_collection := s.Mongo.Rooms

	var existing_room Rooms.Room
	err_find := room_collection.FindOne(context.TODO(), bson.D{{Key: "Name", Value: new_room.Name}}).Decode(&existing_room)

	if err_find == nil {
		return fmt.Errorf("a room with the name '%s' already exists", new_room.Name)
	} else if err_find != mongo.ErrNoDocuments {
		return fmt.Errorf("CreateRoom FindOne error: %s", err_find.Error())
	}

	cursor_last, err_find_last := room_collection.Find(context.TODO(), bson.D{{}}, opt_find_last)

	if err_find_last != nil {
		return fmt.Errorf("CreateRoom Find() error: %s", err_find_last.Error())
	}

	defer func() {
		cursor_last.Close(context.Background())
	}()

	ctx_find_last, cancel_find_last := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel_find_last()

	last_room := &Rooms.Room{}

	for cursor_last.Next(ctx_find_last) {
		err_decode_last := cursor_last.Decode(last_room)

		if err_decode_last != nil {
			log.Println("CreateRoom Decode() error:", cursor_last)
			return err_decode_last
		}
	}

	///// save new curriculm room /////

	save_new_room := new_room
	save_new_room.RoomID = last_room.RoomID + 1

	insert_one_result, err_insert_one := room_collection.InsertOne(context.TODO(), save_new_room)

	if err_insert_one != nil {
		log.Println("CreateRoom: InsertOne() error:", err_insert_one)
		return err_insert_one
	}

	log.Println("CreateRoom: InsertOne Result:", insert_one_result)

	return nil
}

func (s *MongodbWriter) UpdateRoom(updated_room Rooms.Room) error {
	if updated_room.RoomID == 0 {
		return errors.New("error UpdateRoom(): parameter argument missing invalid RoomID")
	}

	if updated_room.Capacity > uint16(Rooms.MAX_ROOM_CAPACITY) {
		return fmt.Errorf("MongoDB.UpdateRoom room to update has a capacity of %d section(s), the maximum allowed is only 15", updated_room.Capacity)
	}

	if s.Mongo.Rooms == nil {
		s.Mongo.Rooms = s.Mongo.Client.Database("gass").Collection("rooms")
	}

	room_collection := s.Mongo.Rooms

	var existing_room Rooms.Room

	err_find := room_collection.FindOne(context.TODO(), bson.D{
		{Key: "Name", Value: updated_room.Name},
		{Key: "RoomID", Value: bson.M{"$ne": updated_room.RoomID}},
	}).Decode(&existing_room)

	if err_find == nil {
		return fmt.Errorf("a room with the name '%s' already exists", updated_room.Name)
	} else if err_find != mongo.ErrNoDocuments {
		return fmt.Errorf("UpdateRoom FindOne error: %s", err_find.Error())
	}

	replace_result, err_replace_one := room_collection.ReplaceOne(
		context.TODO(),
		bson.D{{Key: "RoomID", Value: updated_room.RoomID}},
		updated_room,
	)

	if err_replace_one != nil {
		return fmt.Errorf("UpdateRoom ReplaceOne error: %s", err_replace_one.Error())
	}

	if replace_result.MatchedCount == 0 {
		return fmt.Errorf("no room found with RoomID %d", updated_room.RoomID)
	}

	log.Println("UpdateRoom ReplaceOne result:", replace_result)
	return nil
}

func (s *MongodbWriter) DeleteRoom(room_id uint16) error {

	if room_id == 0 {
		return errors.New("parameter argument missing invalid room ID")
	}

	if s.Mongo.Rooms == nil {
		s.Mongo.Rooms = s.Mongo.Client.Database("gass").Collection("rooms")
	}

	room_collection := s.Mongo.Rooms

	delete_result, err_delete_one := room_collection.DeleteOne(
		context.TODO(),
		bson.D{{Key: "RoomID", Value: room_id}},
	)

	log.Println("UpdateRoom DeleteOne result:", delete_result)

	return err_delete_one
}
