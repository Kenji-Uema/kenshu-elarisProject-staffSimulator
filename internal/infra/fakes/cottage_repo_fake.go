package fakes

import (
	"context"

	"github.com/Kenji-Uema/staffSimulator/internal/domain/documents"
	"github.com/Kenji-Uema/staffSimulator/internal/port"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ port.CottageRepo = (*FakeCottageRepo)(nil)

type FakeCottageRepo struct {
	GetByNameFn            func(ctx context.Context, roomName string) (documents.Cottage, error)
	UpdateCurrentGuestFn   func(ctx context.Context, roomName string, guestID bson.ObjectID) error
	ClearCurrentGuestFn    func(ctx context.Context, roomName string) error
	UpdateCleaningStatusFn func(ctx context.Context, roomName string, cleaningStatus string) error

	GetByNameCallCount            int
	UpdateCurrentGuestCallCount   int
	ClearCurrentGuestCallCount    int
	UpdateCleaningStatusCallCount int

	LastGetByNameCtx             context.Context
	LastGetByNameRoomName        string
	LastUpdateCurrentGuestCtx    context.Context
	LastUpdateRoomName           string
	LastUpdateGuestID            bson.ObjectID
	LastClearCurrentGuestCtx     context.Context
	LastClearRoomName            string
	LastUpdateCleaningStatusCtx  context.Context
	LastUpdateCleaningStatusRoom string
	LastCleaningStatus           string
}

func (f *FakeCottageRepo) GetByName(ctx context.Context, roomName string) (documents.Cottage, error) {
	f.GetByNameCallCount++
	f.LastGetByNameCtx = ctx
	f.LastGetByNameRoomName = roomName

	if f.GetByNameFn != nil {
		return f.GetByNameFn(ctx, roomName)
	}

	return documents.Cottage{}, nil
}

func (f *FakeCottageRepo) UpdateCurrentGuest(ctx context.Context, roomName string, guestID bson.ObjectID) error {
	f.UpdateCurrentGuestCallCount++
	f.LastUpdateCurrentGuestCtx = ctx
	f.LastUpdateRoomName = roomName
	f.LastUpdateGuestID = guestID

	if f.UpdateCurrentGuestFn != nil {
		return f.UpdateCurrentGuestFn(ctx, roomName, guestID)
	}

	return nil
}

func (f *FakeCottageRepo) ClearCurrentGuest(ctx context.Context, roomName string) error {
	f.ClearCurrentGuestCallCount++
	f.LastClearCurrentGuestCtx = ctx
	f.LastClearRoomName = roomName

	if f.ClearCurrentGuestFn != nil {
		return f.ClearCurrentGuestFn(ctx, roomName)
	}

	return nil
}

func (f *FakeCottageRepo) UpdateCleaningStatus(ctx context.Context, roomName string, cleaningStatus string) error {
	f.UpdateCleaningStatusCallCount++
	f.LastUpdateCleaningStatusCtx = ctx
	f.LastUpdateCleaningStatusRoom = roomName
	f.LastCleaningStatus = cleaningStatus

	if f.UpdateCleaningStatusFn != nil {
		return f.UpdateCleaningStatusFn(ctx, roomName, cleaningStatus)
	}

	return nil
}
