package dbErrors

import (
	"fmt"
)

type UnexpectedErr struct {
	Msg string
	Err error
}

func (e *UnexpectedErr) Error() string {
	return fmt.Sprintf("unexpected error, %s: %v", e.Msg, e.Err)
}

type CorruptedDataErr struct {
	Err error
}

func (e CorruptedDataErr) Error() string {
	return fmt.Sprintf("database contains inconsistent cottage data: %v", e.Err)
}

type ErrCottageDoesNotExist struct {
	CottageName string
}

func (e *ErrCottageDoesNotExist) Error() string {
	return fmt.Sprintf("Cottage with name %s does not exist", e.CottageName)
}

type ErrStockDoesNotExist struct {
}

func (e *ErrStockDoesNotExist) Error() string {
	return fmt.Sprintf("Could not find stock collection")
}

type StockInsufficientQuantityErr struct {
	ItemName string
	Quantity int
}

func (e *StockInsufficientQuantityErr) Error() string {
	return fmt.Sprintf("Stock for item %s is insufficient, %d requested", e.ItemName, e.Quantity)
}

type CottageCleaningStatusNotUpdatedErr struct {
	CottageName string
}

func (e *CottageCleaningStatusNotUpdatedErr) Error() string {
	return fmt.Sprintf("could not update cleaningStatus for cottage %s", e.CottageName)
}

type StockItemQuantityNotUpdatedErr struct {
	ItemName string
}

func (e *StockItemQuantityNotUpdatedErr) Error() string {
	return fmt.Sprintf("could not update quantity for item %s", e.ItemName)
}
