package StorageSchedule

import "go.mongodb.org/mongo-driver/mongo"

type MongoDB struct {
	Client    *mongo.Client
	Schedules *mongo.Collection
}
