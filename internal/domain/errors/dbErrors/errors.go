package dbErrors

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type ErrCottageDoesNotExist struct {
	CottageName string
}

func (e *ErrCottageDoesNotExist) Error() string {
	return fmt.Sprintf("Cottage with name %s does not exist", e.CottageName)
}

type ErrGuestDoesNotExist struct {
	Id         bson.ObjectID
	DocumentId string
}

func (e *ErrGuestDoesNotExist) Error() string {
	return fmt.Sprintf("Guest with document id %s does not exist", e.Id.Hex())
}

// ErrBookingRepo indicates an unexpected internal failure in the Booking repository.
var ErrBookingRepo = errors.New("bookingRepository failure")

// ErrCottageRepo indicates an unexpected internal failure in the Cottage repository.
var ErrCottageRepo = errors.New("cottageRepository failure")

var ErrGuestRepo = errors.New("guestRepository failure")

// ErrStockRepo indicates an unexpected internal failure in the Stock repository.
var ErrStockRepo = errors.New("stockRepository failure")
