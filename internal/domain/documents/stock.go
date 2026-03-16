package documents

type Stock struct {
	Id                string
	CleaningItems     Item `bson:"cleaning_items"`
	BathroomAmenities Item `bson:"bathroom_amenities"`
	AromaCandles      Item `bson:"aroma_candles"`
	WaterBottle       Item `bson:"water_bottle"`
	WineBottle        Item `bson:"wine_bottle"`
	TeaBags           Item `bson:"tea_bags"`
	Sweets            Item `bson:"sweets"`
	Chips             Item `bson:"chips"`
}

type Item struct {
	Name     string `bson:"name"`
	Quantity int    `bson:"quantity"`
}
