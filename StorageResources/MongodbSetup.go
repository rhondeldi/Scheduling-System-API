package StorageResources

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDB struct {
	Client               *mongo.Client
	AdminCredentials     *mongo.Collection
	Departments          *mongo.Collection
	Curriculums          *mongo.Collection
	Subjects             *mongo.Collection
	Rooms                *mongo.Collection
	Instructors          *mongo.Collection
	AsyncScheduleRecords *mongo.Collection
}

func NewMongodbClient() *mongo.Client {

	//////////////////////////////////////////////////////////////////////////
	// MongoDB Setup
	//////////////////////////////////////////////////////////////////////////

	log.Println("connecting to MongoDB...")

	log.Printf("MONGODB_CONNECTION_STRING = %s\n", os.Getenv("MONGODB_CONNECTION_STRING"))
	log.Printf("PORT                      = %s\n", os.Getenv("PORT"))

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)

	opts := options.Client().ApplyURI(os.Getenv("MONGODB_CONNECTION_STRING")).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(context.TODO(), opts)

	if err != nil {
		panic(err)
	}

	log.Println("connected to MongoDB...")

	return client
}

func CloseMongodbClient(client *mongo.Client) error {
	err := client.Disconnect(context.TODO())
	log.Print(err)
	return err
}
