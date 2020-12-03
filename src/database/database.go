package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var logErr *log.Logger
var logInfo *log.Logger

func init() {
	logErr = log.New(os.Stderr, "ERROR [database]: ", 1)
	logInfo = log.New(os.Stdout, "INFO [database]: ", 1)
}

func StartDatabase(uri string) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		logErr.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	if err = client.Connect(ctx); err != nil {
		logErr.Fatal(err)
	}
	defer client.Disconnect(ctx)
}
