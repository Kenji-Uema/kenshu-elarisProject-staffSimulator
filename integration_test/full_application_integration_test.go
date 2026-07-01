package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/Kenji-Uema/staffSimulator/integration_test/helpers"
	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
	"google.golang.org/protobuf/types/known/timestamppb"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type scenarioExpectation struct {
	cleaningStatus documents.CleaningStatus
	cleaningItem   int
	soap           int
	wineBottle     int
	aromaCandle    int
	teaBags        int
}

var _ = Describe("Full application integration", Ordered, func() {
	t := GinkgoT()

	BeforeEach(func() {
		resetFixtureState(t)
	})

	It("processes PREPARE_FOR_GUEST end-to-end", func() {
		publishCleaningRequest(t, dto.RequestType_PREPARE_FOR_GUEST)

		Eventually(func(g Gomega) {
			assertScenarioState(g, scenarioExpectation{
				cleaningStatus: documents.CleaningStatusPreparedForGuest,
				cleaningItem:   100,
				soap:           100,
				wineBottle:     99,
				aromaCandle:    99,
				teaBags:        100,
			})
		}, "20s", "250ms").Should(Succeed())
	})

	It("processes DAILY_CLEANING end-to-end", func() {
		publishCleaningRequest(t, dto.RequestType_DAILY_CLEANING)
		publishHourChanges(t, 2*time.Hour)

		Eventually(func(g Gomega) {
			assertScenarioState(g, scenarioExpectation{
				cleaningStatus: documents.CleaningStatusDailyCleaned,
				cleaningItem:   95,
				soap:           90,
				wineBottle:     100,
				aromaCandle:    100,
				teaBags:        100,
			})
		}, "20s", "250ms").Should(Succeed())
	})

	It("processes FULL_CLEANING end-to-end", func() {
		publishCleaningRequest(t, dto.RequestType_FULL_CLEANING)
		publishHourChanges(t, 3*time.Hour, 6*time.Hour)

		Eventually(func(g Gomega) {
			assertScenarioState(g, scenarioExpectation{
				cleaningStatus: documents.CleaningStatusFullyCleaned,
				cleaningItem:   90,
				soap:           70,
				wineBottle:     100,
				aromaCandle:    100,
				teaBags:        100,
			})
		}, "20s", "250ms").Should(Succeed())
	})

	It("processes PREPARE_FOR_SLEEP end-to-end", func() {
		publishCleaningRequest(t, dto.RequestType_PREPARE_FOR_SLEEP)

		Eventually(func(g Gomega) {
			assertScenarioState(g, scenarioExpectation{
				cleaningStatus: documents.CleaningStatusPreparedForSleep,
				cleaningItem:   100,
				soap:           100,
				wineBottle:     100,
				aromaCandle:    99,
				teaBags:        99,
			})
		}, "20s", "250ms").Should(Succeed())
	})

	It("processes immediate restock end-to-end", func() {
		setStockQuantity(t, documents.Soap, 0)
		publishCleaningRequest(t, dto.RequestType_DAILY_CLEANING)
		publishHourChanges(t, 2*time.Hour)

		Eventually(func(g Gomega) {
			assertScenarioState(g, scenarioExpectation{
				cleaningStatus: documents.CleaningStatusDailyCleaned,
				cleaningItem:   95,
				soap:           40,
				wineBottle:     100,
				aromaCandle:    100,
				teaBags:        100,
			})
		}, "20s", "250ms").Should(Succeed())
	})

	It("processes daily restock end-to-end", func() {
		setAllStockQuantities(t, 1)
		publishDayChange(t)

		Eventually(func(g Gomega) {
			var cottage documents.Cottage
			err := suiteCottageCollection.FindOne(context.Background(), bson.M{"name": "A"}).Decode(&cottage)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(cottage.CleaningStatus).To(Equal(documents.CleaningStatusFullyCleaned))

			var stock documents.Stock
			err = suiteStockCollection.FindOne(context.Background(), bson.M{}).Decode(&stock)
			g.Expect(err).NotTo(HaveOccurred())

			restockedItems := 0
			for _, itemName := range documents.ManagedStockItemNames {
				quantity := stock.AsMap()[itemName].Quantity
				g.Expect(quantity == 1 || quantity == 51).To(BeTrue(), "unexpected quantity for %s: %d", itemName, quantity)
				if quantity == 51 {
					restockedItems++
				}
			}
			g.Expect(restockedItems).To(BeNumerically(">", 0))
		}, "20s", "250ms").Should(Succeed())
	})
})

func publishCleaningRequest(t FullGinkgoTInterface, requestType dto.RequestType) {
	t.Helper()

	producer := newProducer(t)
	defer func() {
		Expect(producer.CloseChannel()).To(Succeed())
	}()

	Expect(producer.DeclareExchange(config.ExchangeConfig{Name: suiteCleaningEx, Kind: "direct", AutoDelete: true})).To(Succeed())
	Expect(producer.Publish(context.Background(), &dto.CleaningRequest{
		RoomName: "A",
		Request:  requestType,
	}, "cleaning.request")).To(Succeed())
}

func publishHourChanges(t FullGinkgoTInterface, offsets ...time.Duration) {
	t.Helper()

	producer := newProducer(t)
	defer func() {
		Expect(producer.CloseChannel()).To(Succeed())
	}()

	Expect(producer.DeclareExchange(config.ExchangeConfig{Name: suiteHourEx, Kind: "direct", AutoDelete: true})).To(Succeed())
	for _, offset := range offsets {
		Expect(producer.Publish(context.Background(), &dto.TimeEvent{
			Time: timestamppb.New(time.Now().Add(offset)),
		}, "hour.change")).To(Succeed())
	}
}

func publishDayChange(t FullGinkgoTInterface) {
	t.Helper()

	producer := newProducer(t)
	defer func() {
		Expect(producer.CloseChannel()).To(Succeed())
	}()

	Expect(producer.DeclareExchange(config.ExchangeConfig{Name: suiteDayEx, Kind: "direct", AutoDelete: true})).To(Succeed())
	Expect(producer.Publish(context.Background(), &dto.TimeEvent{
		Time: timestamppb.New(time.Now()),
	}, "day.change")).To(Succeed())
}

func newProducer(t FullGinkgoTInterface) port.MqPublisher {
	t.Helper()

	publisherConn, err := mq.NewRabbitMqConnection(context.Background(), config.RabbitMqConfig{
		Username: "test_user",
		Password: "test_pass",
		Host:     suiteRabbitHost,
		Port:     suiteRabbitPort,
	})
	if err != nil {
		t.Fatalf("create rabbitmq connection: %v", err)
	}
	DeferCleanup(func() {
		Expect(publisherConn.Close()).To(Succeed())
	})

	producer, err := mq.NewRabbitmqProducer(publisherConn, config.PublishConfig{})
	if err != nil {
		t.Fatalf("create producer: %v", err)
	}

	return producer
}

func assertScenarioState(g Gomega, expected scenarioExpectation) {
	var cottage documents.Cottage
	err := suiteCottageCollection.FindOne(context.Background(), bson.M{"name": "A"}).Decode(&cottage)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cottage.CleaningStatus).To(Equal(expected.cleaningStatus))

	var stock documents.Stock
	err = suiteStockCollection.FindOne(context.Background(), bson.M{}).Decode(&stock)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stock.CleaningItems.Quantity).To(Equal(expected.cleaningItem))
	g.Expect(stock.Soap.Quantity).To(Equal(expected.soap))
	g.Expect(stock.WineBottle.Quantity).To(Equal(expected.wineBottle))
	g.Expect(stock.AromaCandles.Quantity).To(Equal(expected.aromaCandle))
	g.Expect(stock.TeaBags.Quantity).To(Equal(expected.teaBags))
}

func resetFixtureState(t FullGinkgoTInterface) {
	t.Helper()

	ctx := context.Background()
	_, err := suiteCottageCollection.DeleteMany(ctx, bson.M{})
	Expect(err).NotTo(HaveOccurred())
	_, err = suiteStockCollection.DeleteMany(ctx, bson.M{})
	Expect(err).NotTo(HaveOccurred())

	cottagesFixturePath, err := helpers.FixturePath("cottages.json")
	if err != nil {
		t.Fatalf("resolve cottages fixture path: %v", err)
	}
	stocksFixturePath, err := helpers.FixturePath("stocks.json")
	if err != nil {
		t.Fatalf("resolve stocks fixture path: %v", err)
	}

	cottages := loadFixture[documents.Cottage](t, cottagesFixturePath)
	stocks := loadFixture[documents.Stock](t, stocksFixturePath)

	cottageDocs := make([]any, len(cottages))
	for i, cottage := range cottages {
		cottageDocs[i] = cottage
	}
	stockDocs := make([]any, len(stocks))
	for i, stock := range stocks {
		stockDocs[i] = stock
	}

	_, err = suiteCottageCollection.InsertMany(ctx, cottageDocs)
	Expect(err).NotTo(HaveOccurred())
	_, err = suiteStockCollection.InsertMany(ctx, stockDocs)
	Expect(err).NotTo(HaveOccurred())
}

func setAllStockQuantities(t FullGinkgoTInterface, quantity int) {
	t.Helper()

	ctx := context.Background()
	var stock documents.Stock
	err := suiteStockCollection.FindOne(ctx, bson.M{}).Decode(&stock)
	if err != nil {
		t.Fatalf("load stock for mutation: %v", err)
	}

	stock.Soap.Quantity = quantity
	stock.CleaningItems.Quantity = quantity
	stock.BathroomAmenities.Quantity = quantity
	stock.AromaCandles.Quantity = quantity
	stock.WaterBottle.Quantity = quantity
	stock.WineBottle.Quantity = quantity
	stock.TeaBags.Quantity = quantity
	stock.Sweets.Quantity = quantity
	stock.Chips.Quantity = quantity

	_, err = suiteStockCollection.ReplaceOne(ctx, bson.M{"_id": stock.Id}, stock)
	if err != nil {
		t.Fatalf("replace stock fixture: %v", err)
	}
}

func setStockQuantity(t FullGinkgoTInterface, itemName string, quantity int) {
	t.Helper()

	ctx := context.Background()
	var stock documents.Stock
	err := suiteStockCollection.FindOne(ctx, bson.M{}).Decode(&stock)
	if err != nil {
		t.Fatalf("load stock for mutation: %v", err)
	}

	switch itemName {
	case documents.CleaningItem:
		stock.CleaningItems.Quantity = quantity
	case documents.Soap:
		stock.Soap.Quantity = quantity
	case documents.BathroomAmenities:
		stock.BathroomAmenities.Quantity = quantity
	case documents.AromaCandle:
		stock.AromaCandles.Quantity = quantity
	case documents.WaterBottle:
		stock.WaterBottle.Quantity = quantity
	case documents.WineBottle:
		stock.WineBottle.Quantity = quantity
	case documents.TeaBags:
		stock.TeaBags.Quantity = quantity
	case documents.Sweets:
		stock.Sweets.Quantity = quantity
	case documents.Chips:
		stock.Chips.Quantity = quantity
	default:
		t.Fatalf("unsupported stock item %q", itemName)
	}

	_, err = suiteStockCollection.ReplaceOne(ctx, bson.M{"_id": stock.Id}, stock)
	if err != nil {
		t.Fatalf("replace stock fixture: %v", err)
	}
}

func loadFixture[T any](t FullGinkgoTInterface, path string) []T {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}

	var docs []T
	if err := json.Unmarshal(data, &docs); err != nil {
		t.Fatalf("unmarshal fixture %q: %v", path, err)
	}

	return docs
}
