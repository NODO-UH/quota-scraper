package database

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var logErr *log.Logger
var logInfo *log.Logger
var globalCollection *mongo.Collection
var currentMonthCollection *mongo.Collection
var qlChan chan *Quotalog

type Quotalog struct {
	DateTime float64
	User     string
	Size     int64
	Url      *url.URL
	From     net.IP
}

func (ql Quotalog) String() string {
	return fmt.Sprintf("DT: %f, User: %s, Size: %d, Url: %s, From: %s", ql.DateTime, ql.User, ql.Size, ql.Url, ql.From)
}

func init() {
	logErr = log.New(os.Stderr, "ERROR [database]: ", 1)
	logInfo = log.New(os.Stdout, "INFO [database]: ", 1)
}

func handler() {
	qlChan = make(chan *Quotalog)
	for {
		ql := <-qlChan
		fmt.Println(ql)
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

	globalCollection = client.Database("quota").Collection("global")
	currentMonthCollection = client.Database("quota").Collection("current_month")

	handler()

	logInfo.Println("Successfully connected and pinged.")
}

func AddQuotalog(ql *Quotalog) {
	qlChan <- ql
}
