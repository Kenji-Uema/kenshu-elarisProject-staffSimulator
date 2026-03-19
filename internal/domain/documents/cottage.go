package documents

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Cottage struct {
	Id             bson.ObjectID   `bson:"_id,omitempty"`
	Name           string          `bson:"name"`
	View           string          `bson:"view"`
	Cleaned        bool            `bson:"cleaned"`
	Details        CottageDetails  `bson:"details"`
	Photos         []string        `bson:"photos"`
	PricePerNight  float32         `bson:"price_per_night"`
	Bookings       []bson.ObjectID `bson:"bookings"`
	CurrentGuest   bson.ObjectID   `bson:"current_guest"`
	CleaningStatus string          `bson:"cleaning_status"`
}

type CottageDetails struct {
	Description          string `bson:"description"`
	View                 string `bson:"view"`
	FurnitureDescription string `bson:"furniture_description"`
	BathroomDescription  string `bson:"bathroom_description"`
	AmenitiesDescription string `bson:"amenities_description"`
}
