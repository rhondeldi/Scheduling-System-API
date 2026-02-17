package StorageResources

import "sync"

type MongodbWriter struct {
	mutex sync.Mutex
	Mongo *MongoDB
}
