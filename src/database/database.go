package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var logErr *log.Logger
var logInfo *log.Logger
var historyCollection *mongo.Collection
var currentMonthCollection *mongo.Collection
var qlChan chan *Quotalog
var UpOk chan bool

type Quotalog struct {
	DateTime float64
	User     string
	Size     int64
	Url      *url.URL
	From     string
}

type QuotaMonth struct {
	User     string
	Max      int64
	Consumed int64
	Enabled  bool
}

func (ql Quotalog) String() string {
	return fmt.Sprintf("DT: %f, User: %s, Size: %d, Url: %s, From: %s", ql.DateTime, ql.User, ql.Size, ql.Url, ql.From)
}

func init() {
	logErr = log.New(os.Stderr, "ERROR [database]: ", 1)
	logInfo = log.New(os.Stdout, "INFO [database]: ", 1)
	UpOk = make(chan bool, 1)
}

func Handler() {
	qlChan = make(chan *Quotalog, 1)
	for {
		// Send log to history
		ql := <-qlChan
		if _, err := historyCollection.InsertOne(context.TODO(), ql); err != nil {
			logErr.Println(err)
		}
		// Update current month
		userMonth := QuotaMonth{}
		filter := bson.M{"user": ql.User}
		if err := currentMonthCollection.FindOne(context.Background(), filter).Decode(&userMonth); err != nil {
			if err == mongo.ErrNoDocuments {
				// User not found in current month
				if _, err := currentMonthCollection.InsertOne(context.TODO(), QuotaMonth{
					User:     ql.User,
					Max:      8000000,
					Consumed: ql.Size,
					Enabled:  true,
				}); err != nil {
					logErr.Println(err)
				}
				// TODO: Check if Consumed > Max
			}
		} else if userMonth.Enabled {
			// User found and quota enabled
			_, err := currentMonthCollection.UpdateOne(
				context.Background(),
				bson.M{"user": userMonth.User},
				bson.D{
					{"$set", bson.D{{"consumed", userMonth.Consumed + ql.Size}}},
				},
			)
			if err != nil {
				logErr.Println(err)
			}
		}
	}
}

func StartDatabase(uri string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))

	if err != nil {
		logErr.Fatal(err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			logErr.Fatal(err)
		}
	}()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		logErr.Fatal(err)
	}

	historyCollection = client.Database("quota").Collection("history")
	currentMonthCollection = client.Database("quota").Collection("current_month")

	logInfo.Println("Successfully connected and pinged.")

	UpOk <- true

	Handler()

}

func AddQuotalog(ql *Quotalog) {
	qlChan <- ql
}
