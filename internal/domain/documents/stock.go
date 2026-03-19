package documents

import "go.mongodb.org/mongo-driver/v2/bson"

const (
	CleaningItem      string = "cleaningItem"
	BathroomAmenities string = "bathroomAmenities"
	AromaCandle       string = "aromaCandle"
	WaterBottle       string = "waterBottle"
	WineBottle        string = "wineBottle"
	TeaBags           string = "teaBags"
	Sweets            string = "sweets"
	Chips             string = "chips"
)

var ManagedStockItemNames = []string{
	CleaningItem,
	BathroomAmenities,
	AromaCandle,
	WaterBottle,
	WineBottle,
	TeaBags,
	Sweets,
	Chips,
}

type Stock struct {
	Id                bson.ObjectID `bson:"_id,omitempty"`
	CleaningItems     StockItem     `bson:"cleaning_items"`
	BathroomAmenities StockItem     `bson:"bathroom_amenities"`
	AromaCandles      StockItem     `bson:"aroma_candles"`
	WaterBottle       StockItem     `bson:"water_bottle"`
	WineBottle        StockItem     `bson:"wine_bottle"`
	TeaBags           StockItem     `bson:"tea_bags"`
	Sweets            StockItem     `bson:"sweets"`
	Chips             StockItem     `bson:"chips"`
}

type StockItem struct {
	Name     string `bson:"name"`
	Quantity int    `bson:"quantity"`
}

func (s *Stock) AsMap() map[string]StockItem {
	return map[string]StockItem{
		CleaningItem:      s.CleaningItems,
		BathroomAmenities: s.BathroomAmenities,
		AromaCandle:       s.AromaCandles,
		WaterBottle:       s.WaterBottle,
		WineBottle:        s.WineBottle,
		TeaBags:           s.TeaBags,
		Sweets:            s.Sweets,
		Chips:             s.Chips,
	}
}
