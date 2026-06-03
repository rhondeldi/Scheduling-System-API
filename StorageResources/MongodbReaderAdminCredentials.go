package StorageResources

import (
	"context"
	"log"

	AdminResource "github.com/mrdcvlsc/scheduling-system-backend/Resources/Admin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const adminCredentialsCollectionName = "admin_credentials"
const adminCredentialsDocumentID = "admin_credentials"

func (s *MongodbReader) ReadAdminCredentials() (*AdminResource.AdminCredentials, error) {
	if s.Mongo.AdminCredentials == nil {
		s.Mongo.AdminCredentials = s.Mongo.Client.Database("gass").Collection(adminCredentialsCollectionName)
	}

	credentials := &AdminResource.AdminCredentials{}
	opts := options.FindOne().SetProjection(bson.D{{Key: "_id", Value: 0}})

	err := s.Mongo.AdminCredentials.FindOne(
		context.TODO(),
		bson.D{{Key: "_id", Value: adminCredentialsDocumentID}},
		opts,
	).Decode(credentials)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		log.Print(err)
		return nil, err
	}

	return credentials, nil
}
