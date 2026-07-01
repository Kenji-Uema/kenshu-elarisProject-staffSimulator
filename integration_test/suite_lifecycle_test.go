package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Kenji-Uema/staffSimulator/integration_test/helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	suiteMongoHost         string
	suiteRabbitHost        string
	suiteRabbitPort        int
	suiteClockHost         string
	suiteClockPort         int
	suiteAppPort           int
	suiteStopMain          func()
	suiteRunErrCh          <-chan error
	suiteReady             bool
	suiteDBName            string
	suiteCleaningEx        string
	suiteDayEx             string
	suiteHourEx            string
	suiteRawMongoClient    *mongo.Client
	suiteCottageCollection *mongo.Collection
	suiteStockCollection   *mongo.Collection
	suiteRawRabbitConn     *amqp.Connection
	suiteRawRabbitChannel  *amqp.Channel
)

var _ = BeforeSuite(func() {
	t := GinkgoT()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	DeferCleanup(cancel)

	suiteMongoHost, suiteRabbitHost, suiteRabbitPort = helpers.StartContainers(ctx)
	suiteDBName = "test_db"
	suiteCleaningEx = "ex.cleaning.request"
	suiteDayEx = "ex.day_change.event"
	suiteHourEx = "ex.hour_change.event"

	cottagesFixturePath, err := helpers.FixturePath("cottages.json")
	if err != nil {
		Fail(fmt.Sprintf("resolve cottages fixture path: %v", err))
	}
	stocksFixturePath, err := helpers.FixturePath("stocks.json")
	if err != nil {
		Fail(fmt.Sprintf("resolve stocks fixture path: %v", err))
	}

	helpers.SeedMongoFromFixtures(
		t,
		suiteMongoHost,
		suiteDBName,
		cottagesFixturePath,
		stocksFixturePath,
	)
	suiteClockHost, suiteClockPort = helpers.StartClockEmulator(t)

	suiteRawMongoClient, err = mongo.Connect(options.Client().ApplyURI(fmt.Sprintf("mongodb://test_user:test_pass@%s", suiteMongoHost)))
	if err != nil {
		Fail(fmt.Sprintf("connect mongo: %v", err))
	}

	db := suiteRawMongoClient.Database(suiteDBName)
	suiteCottageCollection = db.Collection("cottage")
	suiteStockCollection = db.Collection("Stock")

	suiteRawRabbitConn, err = amqp.Dial(amqp.URI{
		Scheme:   "amqp",
		Username: "test_user",
		Password: "test_pass",
		Host:     suiteRabbitHost,
		Port:     suiteRabbitPort,
	}.String())
	if err != nil {
		Fail(fmt.Sprintf("dial rabbitmq: %v", err))
	}

	suiteRawRabbitChannel, err = suiteRawRabbitConn.Channel()
	if err != nil {
		Fail(fmt.Sprintf("open rabbitmq channel: %v", err))
	}

	Expect(suiteRawRabbitChannel.ExchangeDeclare(suiteCleaningEx, "direct", false, true, false, false, nil)).To(Succeed())
	Expect(suiteRawRabbitChannel.ExchangeDeclare(suiteDayEx, "direct", false, true, false, false, nil)).To(Succeed())
	Expect(suiteRawRabbitChannel.ExchangeDeclare(suiteHourEx, "direct", false, true, false, false, nil)).To(Succeed())

	suiteAppPort = helpers.FreeTCPPort(t)
	suiteStopMain, suiteRunErrCh = helpers.ApplicationStart(
		t,
		helpers.ApplicationConfig{
			AppPort:          suiteAppPort,
			ClockHost:        suiteClockHost,
			ClockPort:        fmt.Sprintf("%d", suiteClockPort),
			MongoURI:         fmt.Sprintf("mongodb://test_user:test_pass@%s", suiteMongoHost),
			MongoDatabase:    suiteDBName,
			RabbitHost:       suiteRabbitHost,
			RabbitPort:       suiteRabbitPort,
			CleaningExchange: suiteCleaningEx,
			DayExchange:      suiteDayEx,
			HourExchange:     suiteHourEx,
		},
	)

	if err := waitForHTTP200OrExit(suiteAppPort, suiteRunErrCh, 30*time.Second); err != nil {
		Fail(fmt.Sprintf("integration app failed to become ready: %v", err))
	}

	suiteReady = true
})

var _ = AfterSuite(func() {
	if !suiteReady || suiteStopMain == nil {
		return
	}

	suiteStopMain()
	select {
	case err := <-suiteRunErrCh:
		if err != nil {
			Fail(fmt.Sprintf("main returned error: %v", err))
		}
	case <-time.After(45 * time.Second):
		Fail("main did not stop after context cancel")
	}

	if suiteRawRabbitChannel != nil {
		Expect(suiteRawRabbitChannel.Close()).To(Succeed())
	}
	if suiteRawRabbitConn != nil {
		Expect(suiteRawRabbitConn.Close()).To(Succeed())
	}
	if suiteRawMongoClient != nil {
		Expect(suiteRawMongoClient.Database(suiteDBName).Drop(context.Background())).To(Succeed())
		Expect(suiteRawMongoClient.Disconnect(context.Background())).To(Succeed())
	}
})

func waitForHTTP200OrExit(appPort int, runErrCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://127.0.0.1:%d/readyz", appPort)
	client := &http.Client{Timeout: 2 * time.Second}

	for {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case runErr := <-runErrCh:
			if runErr != nil {
				return fmt.Errorf("application exited before ready: %w", runErr)
			}
			return fmt.Errorf("application exited before ready")
		default:
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to return 200", url)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
