package mdb

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/config"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/dbErrors"
	"github.com/Kenji-Uema/staffSimulator/internal/port"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type cottageRepo struct {
	collection *mongo.Collection
}

func NewCottageRepo(db *mongo.Database, config config.CottageCollectionConfig) port.CottageRepo {
	return &cottageRepo{collection: db.Collection(config.Name)}
}

func (r *cottageRepo) GetByName(ctx context.Context, roomName string) (documents.Cottage, error) {
	if err := validation.New().NotBlank("roomName", roomName).Validate(); err != nil {
		return documents.Cottage{}, err
	}

	filter := bson.M{"name": roomName}

	var cottage documents.Cottage
	if err := r.collection.FindOne(ctx, filter).Decode(&cottage); err != nil {
		return documents.Cottage{}, fmt.Errorf("%w: could not find cottage by name, roomName=%s: %v",
			dbErrors.ErrCottageRepo, roomName, err)
	}

	return cottage, nil
}

func (r *cottageRepo) UpdateCurrentGuest(ctx context.Context, roomName string, guestId bson.ObjectID) error {
	if err := validation.New().NotBlank("roomName", roomName).NotNilObjectID("guestId", guestId).Validate(); err != nil {
		return err
	}

	filter := bson.M{"name": roomName}
	update := bson.M{"$set": bson.M{"current_guest": guestId}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%w: could not update current guest for the room, roomName=%s guestId=%s: %v",
			dbErrors.ErrCottageRepo, roomName, guestId.Hex(), err)
	}

	if result.MatchedCount == 0 {
		return &dbErrors.ErrCottageDoesNotExist{CottageName: roomName}
	}

	return nil
}

func (r *cottageRepo) ClearCurrentGuest(ctx context.Context, roomName string) error {
	if err := validation.New().NotBlank("roomName", roomName).Validate(); err != nil {
		return err
	}

	filter := bson.M{"name": roomName}
	update := bson.M{"$set": bson.M{"current_guest": bson.NilObjectID}}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("%w: could not clear current guest for the room, roomName=%s: %v",
			dbErrors.ErrCottageRepo, roomName, err)
	}

	if result.MatchedCount == 0 {
		return &dbErrors.ErrCottageDoesNotExist{CottageName: roomName}
	}

	return nil
}
