package mongodb

import (
	"context"
	"grpcmongoproject/pkg/utils"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateMongoClient() (*mongo.Client, error) {
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Println("Error connecting to MongoDb: ", err)
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		log.Fatalln("Unable to ping mongodb server: ", err)
		return nil, utils.ErrorHandler(err, "Unable to ping mongodb server.")
	}
	log.Println("Connected to mongodb")
	return mongoClient, nil
}
