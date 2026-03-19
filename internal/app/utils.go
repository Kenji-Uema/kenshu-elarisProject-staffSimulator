package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/app/validation"
	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/errors/validationErrors"
	"google.golang.org/protobuf/proto"
)

func unmarshalTimeEvent(ctx context.Context, body []byte) (time.Time, error) {
	var timeEvent dto.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		slog.WarnContext(ctx, "invalid day.changed payload", "error", err)
		return time.Time{}, err

	}

	if err := validation.New().NotZeroValue("time", timeEvent.GetTime()).Validate(); err != nil {
		return time.Time{}, err
	}

	return timeEvent.GetTime().AsTime(), nil
}

func UnmarshalCleaningRequest(ctx context.Context, body []byte) (domain.CleaningRequest, error) {
	var cleaningRequest dto.CleaningRequest
	if err := proto.Unmarshal(body, &cleaningRequest); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshalCleaningRequest cleaning request", "error", err)
		return domain.CleaningRequest{}, err
	}

	if err := validation.New().
		NotBlank("roomName", cleaningRequest.GetRoomName()).Validate(); err != nil {

		return domain.CleaningRequest{}, err
	}

	if cleaningRequest.GetRequest() == dto.RequestType_UNSPECIFIED {
		return domain.CleaningRequest{}, &validationErrors.ErrValidationConstrain{
			Field:   "request",
			Message: "must not be unspecified",
		}
	}

	return domain.CleaningRequest{
		RoomName:    cleaningRequest.GetRoomName(),
		RequestType: cleaningRequest.GetRequest().String(),
	}, nil
}
