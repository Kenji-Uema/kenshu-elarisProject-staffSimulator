package mdb

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/port"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type cottageRepo struct {
	collection *mongo.Collection
}

func NewCottageRepo(db *mongo.Database) port.CottageRepo {
	return &cottageRepo{collection: db.Collection("cottage")}
}

func (r *cottageRepo) UpdateCleaningStatus(ctx context.Context, cottageName string, cleaningStatus string) error {
	if err := validation.New().NotBlank("cottageName", cottageName).NotBlank("cleaningStatus", cleaningStatus).Validate(); err != nil {
		return err
	}

	filter := bson.M{"name": cottageName}
	update := bson.M{"$set": bson.M{"cleaning_status": cleaningStatus}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return &dbErrors.UnexpectedErr{Msg: "failed to update cottage clean status", Err: err}
	}

	if result.MatchedCount == 0 {
		return &dbErrors.ErrCottageDoesNotExist{CottageName: cottageName}
	}

	if result.ModifiedCount == 0 {
		return &dbErrors.CottageCleaningStatusNotUpdatedErr{CottageName: cottageName}
	}

	return nil
}
