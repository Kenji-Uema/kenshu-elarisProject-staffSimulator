package documents

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type CleaningStatus string
type KeyHolder string

const (
	CleaningStatusPreparedForGuest CleaningStatus = "prepared_for_guest"
	CleaningStatusDailyCleaned     CleaningStatus = "daily_cleaned"
	CleaningStatusPreparedForSleep CleaningStatus = "prepared_for_sleep"
	CleaningStatusFullyCleaned     CleaningStatus = "fully_cleaned"
)

type Cottage struct {
	Id             bson.ObjectID   `bson:"_id,omitempty"`
	Name           string          `bson:"name"`
	View           string          `bson:"view"`
	Details        CottageDetails  `bson:"details"`
	Photos         []string        `bson:"photos"`
	PricePerNight  float32         `bson:"price_per_night"`
	Bookings       []bson.ObjectID `bson:"bookings"`
	CurrentGuest   bson.ObjectID   `bson:"current_guest"`
	CleaningStatus CleaningStatus  `bson:"cleaning_status"`
	Key            Key             `bson:"key"`
}

type CottageDetails struct {
	Description          string `bson:"description"`
	View                 string `bson:"view"`
	FurnitureDescription string `bson:"furniture_description"`
	BathroomDescription  string `bson:"bathroom_description"`
	AmenitiesDescription string `bson:"amenities_description"`
}

type Key struct {
	Number string    `bson:"number"`
	Holder KeyHolder `bson:"holder"`
}
