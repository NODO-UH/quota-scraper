package database

import (
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/NODO-UH/quota-scraper/src/squid"
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
var freeCollection *mongo.Collection
var qlChan chan *Quotalog
var UpOk chan bool
var freeRegexp *regexp.Regexp

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

type QuotaFree struct {
	Regex string
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

func UpdateCurrentMonth(ql *Quotalog, scraperId string) error {
	if IsFree(ql.Url) {
		return nil
	}
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

				// Cut in squid file
				squid.Cut(ql.User)

				// Set Enabled to false
				currentMonthCollection.FindOneAndUpdate(
					context.Background(),
					bson.M{"user": ql.User},
					bson.D{
						{"$set", bson.D{{"enabled", false}}},
						{"$set", bson.D{{"cutter", scraperId}}},
					})
			}
		}

	} else if err == mongo.ErrNoDocuments {
		// User not found in current_month
		logErr.Printf("unkown user %s in current_month\n", ql.User)
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
			UpdateCurrentMonth(ql, scraperId)
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
	freeCollection = client.Database("quota").Collection("free")

	logInfo.Println("Successfully connected and pinged.")

	UpOk <- true

	Handler(scraperId)
}

func AddQuotalog(ql *Quotalog) {
	qlChan <- ql
}

func GetFreeRegex() []string {
	freeRegexs := []QuotaFree{}
	if cursor, err := freeCollection.Find(context.TODO(), bson.M{}); err != nil {
		logErr.Println(err)
	} else {
		if err = cursor.All(context.TODO(), &freeRegexs); err != nil {
			logErr.Println(err)
		}
	}
	regexs := []string{}
	for _, r := range freeRegexs {
		regexs = append(regexs, r.Regex)
	}
	return regexs
}

func LoadFree() {
	// Get all regexs
	regexs := GetFreeRegex()
	if len(regexs) == 0 {
		logInfo.Println("not free regexs")
		freeRegexp, _ = regexp.Compile("-")
	} else {
		// Join all regex
		joinRegex := strings.Join(regexs, "|")
		var err error
		if freeRegexp, err = regexp.Compile(joinRegex); err != nil {
			logErr.Println(err)
		} else {
			logInfo.Println("free regex builded")
		}
	}
}

func IsFree(domain string) bool {
	return freeRegexp.MatchString(domain)
}
