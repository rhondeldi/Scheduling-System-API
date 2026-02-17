package StorageResources

import (
	"context"
	"log"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbReader) ReadAllRooms() ([]Rooms.Room, error) {

	if s.Mongo.Rooms == nil {
		s.Mongo.Rooms = s.Mongo.Client.Database("gass").Collection("rooms")
	}

	room_collection := s.Mongo.Rooms

	find_opts := options.Find().SetSort(bson.D{{Key: "RoomID", Value: 1}}).SetProjection(bson.D{{Key: "_id", Value: 0}})

	cursor, err := room_collection.Find(context.TODO(), bson.D{{}}, find_opts)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	defer func() {
		cursor.Close(context.Background())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rooms := make([]Rooms.Room, 0)

	for cursor.Next(ctx) {
		room := &Rooms.Room{}

		err := cursor.Decode(room)

		if err != nil {
			log.Println("ReadAllRooms: cursor.Next() error:")
			panic(err)
		}

		rooms = append(rooms, *room)
	}

	return rooms, nil
}

func (s *MongodbReader) ReadRoom(room_id uint16) (*Rooms.Room, error) {

	if s.Mongo.Rooms == nil {
		s.Mongo.Rooms = s.Mongo.Client.Database("gass").Collection("rooms")
	}

	room_collection := s.Mongo.Rooms

	room := &Rooms.Room{}

	filter := bson.D{{Key: "RoomID", Value: room_id}}
	opts := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	err := room_collection.FindOne(context.TODO(), filter, opts).Decode(room)

	if err != nil {
		log.Print(err)
		return nil, err
	}

	return room, nil
}
