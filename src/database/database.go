package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/NODO-UH/quota-scraper/src/configuration"

	slog "github.com/NODO-UH/quota-scraper/src/log"
	"github.com/NODO-UH/quota-scraper/src/squid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var dbStatus *mongo.Collection
var historyCollection *mongo.Collection
var currentMonthCollection *mongo.Collection
var freeCollection *mongo.Collection
var config configuration.ScraperConfig
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

type FreeItem struct {
	Regex *string
}

func (ql Quotalog) String() string {
	return fmt.Sprintf("DT: %f, User: %s, Size: %d, Url: %s, From: %s", ql.DateTime, ql.User, ql.Size, ql.Url, ql.From)
}

func init() {
	UpOk = make(chan bool, 1)
}

func logInfo(message string) {
	slog.Info(message, "[database]")
}

func logError(message string) {
	slog.Err(message, "[database]")
}

func logFatal(message string) {
	logError(message)
	os.Exit(1)
}

func AddHistory(ql *Quotalog) error {
	if _, err := historyCollection.InsertOne(context.TODO(), ql); err != nil {
		logError(err.Error())
		return err
	}
	return nil
}

func UpdateLastDateTime(lastDateTime float64) error {
	if _, err := dbStatus.UpdateOne(
		context.Background(),
		bson.M{"prop": "lastDateTime", "scraperid": *config.Id},
		bson.D{
			{"$set", bson.D{{"value", lastDateTime}}},
		},
	); err != nil {
		logError(err.Error())
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
			logError(err.Error())
			return err
		} else {
			// Check for cut
			if userMonth.Enabled && userMonth.Consumed+ql.Size > userMonth.Max {
				logInfo(fmt.Sprintf("CUT %s", userMonth.User))

				// Cut in squid file
				if err := squid.Cut(ql.User); err == nil {
					// Set Enabled to false
					currentMonthCollection.FindOneAndUpdate(
						context.Background(),
						bson.M{"user": ql.User},
						bson.D{
							{"$set", bson.D{{"enabled", false}}},
							{"$set", bson.D{{"cutter", config.Id}}},
						})
				}
			}
		}

	} else if err == mongo.ErrNoDocuments {
		// User not found in current_month
		logError(fmt.Sprintf("unkown user %s in current_month\n", ql.User))
	} else {
		// Unexpected error
		logError(err.Error())
		return err
	}
	return nil
}

func Handler() {
	qlChan = make(chan *Quotalog, 1)
	for {
		// Wait for new QuotaLog
		ql := <-qlChan

		// Send log to history
		if err := AddHistory(ql); err == nil {
			// Update LastDateTime
			UpdateLastDateTime(ql.DateTime)

			// Update current month
			UpdateCurrentMonth(ql)
		}
	}
}

func GetLastDateTime() float64 {
	lastDateTime := DBProperty{}
	// Load db status
	if err := dbStatus.FindOne(context.Background(), bson.M{"prop": "lastDateTime", "scraperid": *config.Id}).Decode(&lastDateTime); err != nil {
		if err == mongo.ErrNoDocuments {
			lastDateTime = DBProperty{
				ScraperId: *config.Id,
				Prop:      "lastDateTime",
				Value:     0,
			}
			// Create db status
			if _, err := dbStatus.InsertOne(context.TODO(), lastDateTime); err != nil {
				logError(err.Error())
			}
		} else {
			logError(err.Error())
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

func StartDatabase() {
	config = configuration.GetConfiguration()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(*config.DbUri))

	if err != nil {
		logFatal(err.Error())
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			logFatal(err.Error())
		}
	}()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		logFatal(err.Error())
	}

	dbStatus = client.Database("quota").Collection("status")
	historyCollection = client.Database("quota").Collection("history")
	currentMonthCollection = client.Database("quota").Collection("current_month")
	freeCollection = client.Database("quota").Collection("free")

	logInfo("Successfully connected and pinged.")

	UpOk <- true

	Handler()
}

func GetAllFree() []FreeItem {
	free := []FreeItem{}
	if cur, err := freeCollection.Find(context.Background(), bson.D{}); err != nil {
		logError(err.Error())
	} else if err = cur.All(context.TODO(), &free); err != nil {
		logError(err.Error())
	}
	return free
}

func AddQuotalog(ql *Quotalog) {
	qlChan <- ql
}
