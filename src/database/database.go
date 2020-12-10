package database

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var logErr *log.Logger
var logInfo *log.Logger
var dbStatus *mongo.Collection
var historyCollection *mongo.Collection
var currentMonthCollection *mongo.Collection
var qlChan chan *Quotalog
var UpOk chan bool

type Quotalog struct {
	DateTime float64
	User     string
	Size     int64
	Url      string
	From     string
}

type QuotaMonth struct {
	User     string
	Max      int64
	Consumed int64
	Enabled  bool
}

type DBProperty struct {
	ScraperId string
	Prop      string
	Value     interface{}
}

func (ql Quotalog) String() string {
	return fmt.Sprintf("DT: %f, User: %s, Size: %d, Url: %s, From: %s", ql.DateTime, ql.User, ql.Size, ql.Url, ql.From)
}

func init() {
	UpOk = make(chan bool, 1)
}

func SetLogOutput(w io.Writer) {
	logErr = log.New(w, "ERROR [database]: ", log.LstdFlags|log.Lmsgprefix)
	logInfo = log.New(w, "INFO [database]: ", log.LstdFlags|log.Lmsgprefix)
}

func AddHistory(ql *Quotalog) error {
	if _, err := historyCollection.InsertOne(context.TODO(), ql); err != nil {
		logErr.Println(err)
		return err
	}
	return nil
}

func UpdateLastDateTime(lastDateTime float64, scraperId string) error {
	if _, err := dbStatus.UpdateOne(
		context.Background(),
		bson.M{"prop": "lastDateTime", "scraperid": scraperId},
		bson.D{
			{"$set", bson.D{{"value", lastDateTime}}},
		},
	); err != nil {
		logErr.Println(err)
		return err
	}
	return nil
}

func UpdateCurrentMonth(ql *Quotalog) error {
	result := currentMonthCollection.FindOneAndUpdate(
		context.Background(),
		bson.M{"user": ql.User},
		bson.D{
			{"$inc", bson.D{{"consumed", ql.Size}}},
		})

	err := result.Err()
	if err == nil {
		userMonth := QuotaMonth{}
		if err := result.Decode(&userMonth); err != nil {
			logErr.Println(err)
			return err
		} else {

			// Check for cut
			if userMonth.Enabled && userMonth.Consumed+ql.Size > userMonth.Max {
				logInfo.Printf("CUT %s", userMonth.User)

				// TODO: Cut in squid file

				// Set Enabled to false
				currentMonthCollection.FindOneAndUpdate(
					context.Background(),
					bson.M{"user": ql.User},
					bson.D{
						{"$set", bson.D{{"enabled", false}}},
					})
			}
		}

	} else if err == mongo.ErrNoDocuments {
		// Not exists => Insert new in current month
		if _, err := currentMonthCollection.InsertOne(context.TODO(), QuotaMonth{
			User:     ql.User,
			Max:      50000,
			Consumed: ql.Size,
			Enabled:  true,
		}); err != nil {
			logErr.Println(err)
			return err
		}
	} else {
		// Unexpected error
		logErr.Println(err)
		return err
	}
	return nil
}

func Handler(scraperId string) {
	qlChan = make(chan *Quotalog, 1)
	for {
		// Wait for new QuotaLog
		ql := <-qlChan

		// Send log to history
		if err := AddHistory(ql); err == nil {
			// Update LastDateTime
			UpdateLastDateTime(ql.DateTime, scraperId)

			// Update current month
			UpdateCurrentMonth(ql)
		}
	}
}

func GetLastDateTime(scraperId string) float64 {
	lastDateTime := DBProperty{}
	// Load db status
	if err := dbStatus.FindOne(context.Background(), bson.M{"prop": "lastDateTime", "scraperid": scraperId}).Decode(&lastDateTime); err != nil {
		if err == mongo.ErrNoDocuments {
			lastDateTime = DBProperty{
				ScraperId: scraperId,
				Prop:      "lastDateTime",
				Value:     0,
			}
			// Create db status
			if _, err := dbStatus.InsertOne(context.TODO(), lastDateTime); err != nil {
				logErr.Println(err)
			}
		} else {
			logErr.Println(err)
		}
		return 0
	}
	switch t := lastDateTime.Value.(type) {
	case float64:
		return t
	default:
		return 0
	}
}

func StartDatabase(uri string, scraperId string) {
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

	dbStatus = client.Database("quota").Collection("status")
	historyCollection = client.Database("quota").Collection("history")
	currentMonthCollection = client.Database("quota").Collection("current_month")

	logInfo.Println("Successfully connected and pinged.")

	UpOk <- true

	Handler(scraperId)

}

func AddQuotalog(ql *Quotalog) {
	qlChan <- ql
}
