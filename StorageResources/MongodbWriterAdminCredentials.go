package StorageResources

import (
	"context"
	"log"

	AdminResource "github.com/mrdcvlsc/scheduling-system-backend/Resources/Admin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *MongodbWriter) UpsertAdminCredentials(credentials AdminResource.AdminCredentials) error {
	if s.Mongo.AdminCredentials == nil {
		s.Mongo.AdminCredentials = s.Mongo.Client.Database("gass").Collection(adminCredentialsCollectionName)
	}

	filter := bson.D{{Key: "_id", Value: adminCredentialsDocumentID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "username", Value: credentials.Username},
			{Key: "passwordHash", Value: credentials.PasswordHash},
		}},
		{Key: "$unset", Value: bson.D{
			{Key: "password", Value: ""},
		}},
	}

	result, err := s.Mongo.AdminCredentials.UpdateOne(
		context.TODO(),
		filter,
		update,
		options.Update().SetUpsert(true),
	)

	if err != nil {
		log.Print(err)
		return err
	}

	log.Println("UpsertAdminCredentials UpdateOne result:", result)
	return nil
}
