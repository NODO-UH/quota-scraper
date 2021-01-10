package database

import (
	"context"
	"fmt"
	"time"

	"github.com/NODO-UH/quota-scraper/src/configuration"
	log "github.com/NODO-UH/quota-scraper/src/log"

	"github.com/NODO-UH/quota-scraper/src/squid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	// DB collections
	dbStatus               *mongo.Collection
	historyCollection      *mongo.Collection
	currentMonthCollection *mongo.Collection
	freeCollection         *mongo.Collection
	// Configuration
	config configuration.ScraperConfig
	// Channel for QuotaLog
	qlChan chan *Quotalog
	// Channel to send Ok when StartDatabase ends
	UpOk chan bool
)

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

func AddHistory(ql *Quotalog) error {
	if _, err := historyCollection.InsertOne(context.TODO(), ql); err != nil {
		log.Error.Println(err)
		return err
	}
	return nil
}

func UpdateLastDateTime(lastDateTime float64) {
	if _, err := dbStatus.UpdateOne(
		context.Background(),
		bson.M{"prop": "lastDateTime", "scraperid": *config.Id},
		bson.D{
			{"$set", bson.D{{"value", lastDateTime}}},
		},
	); err != nil {
		log.Error.Println(err)
	}
}

func UpdateCurrentMonth(ql *Quotalog) {
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
			log.Error.Println(err)
		} else {
			// Check for cut
			if userMonth.Enabled && userMonth.Consumed+ql.Size > userMonth.Max {
				log.Info.Printf("CUT %s\n", userMonth.User)

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
		log.Error.Printf("unkown user %s in current_month\n", ql.User)
	} else {
		// Unexpected error
		log.Error.Println(err)
	}
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
				log.Error.Println(err)
			}
		} else {
			log.Error.Println(err)
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
		log.Error.Fatalln(err)
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Error.Fatalln(err)
		}
	}()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Error.Fatalln(err)
	}

	dbStatus = client.Database("quota").Collection("status")
	historyCollection = client.Database("quota").Collection("history")
	currentMonthCollection = client.Database("quota").Collection("current_month")
	freeCollection = client.Database("quota").Collection("free")

	log.Info.Println("Successfully connected and pinged.")

	UpOk <- true

	Handler()
}

func GetAllFree() []FreeItem {
	free := []FreeItem{}
	if cur, err := freeCollection.Find(context.Background(), bson.D{}); err != nil {
		log.Error.Println(err)
	} else if err = cur.All(context.TODO(), &free); err != nil {
		log.Error.Println(err)
	}
	return free
}

func AddQuotalog(ql *Quotalog) {
	qlChan <- ql
}
