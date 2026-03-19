package app

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/staffSimulator/internal/domain"
	"github.com/Kenji-Uema/staffSimulator/internal/domain/dto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUnmarshalTimeEvent(t *testing.T) {
	t.Parallel()

	t.Run("returns parsed time for valid payload", func(t *testing.T) {
		t.Parallel()

		want := time.Date(2026, 3, 18, 15, 30, 0, 0, time.UTC)
		body := mustMarshalTestTimeEvent(t, &dto.TimeEvent{
			Time: timestamppb.New(want),
		})

		got, err := unmarshalTimeEvent(context.Background(), body)
		if err != nil {
			t.Fatalf("unmarshalTimeEvent() error = %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("unmarshalTimeEvent() = %v, want %v", got, want)
		}
	})

	t.Run("returns error for invalid payload", func(t *testing.T) {
		t.Parallel()

		_, err := unmarshalTimeEvent(context.Background(), []byte("not-protobuf"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when time is missing", func(t *testing.T) {
		t.Parallel()

		body := mustMarshalTestTimeEvent(t, &dto.TimeEvent{})

		_, err := unmarshalTimeEvent(context.Background(), body)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "time has violations; must not be zero value" {
			t.Fatalf("error = %q, want %q", err.Error(), "time has violations; must not be zero value")
		}
	})
}

func TestUnmarshalCleaningRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns parsed cleaning request for valid payload", func(t *testing.T) {
		t.Parallel()

		body := mustMarshalTestCleaningRequest(t, &dto.CleaningRequest{
			RoomName: "A",
			Request:  dto.RequestType_DAILY_CLEANING,
		})

		got, err := UnmarshalCleaningRequest(context.Background(), body)
		if err != nil {
			t.Fatalf("UnmarshalCleaningRequest() error = %v", err)
		}

		want := domain.CleaningRequest{
			RoomName:    "A",
			RequestType: "DAILY_CLEANING",
		}
		if got != want {
			t.Fatalf("UnmarshalCleaningRequest() = %+v, want %+v", got, want)
		}
	})

	t.Run("returns error for invalid payload", func(t *testing.T) {
		t.Parallel()

		_, err := UnmarshalCleaningRequest(context.Background(), []byte("not-protobuf"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when room name is blank", func(t *testing.T) {
		t.Parallel()

		body := mustMarshalTestCleaningRequest(t, &dto.CleaningRequest{
			RoomName: "",
			Request:  dto.RequestType_FULL_CLEANING,
		})

		_, err := UnmarshalCleaningRequest(context.Background(), body)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when request is unspecified", func(t *testing.T) {
		t.Parallel()

		body := mustMarshalTestCleaningRequest(t, &dto.CleaningRequest{
			RoomName: "A",
			Request:  dto.RequestType_UNSPECIFIED,
		})

		_, err := UnmarshalCleaningRequest(context.Background(), body)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func mustMarshalTestTimeEvent(t *testing.T, event *dto.TimeEvent) []byte {
	t.Helper()

	body, err := proto.Marshal(event)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return body
}

func mustMarshalTestCleaningRequest(t *testing.T, request *dto.CleaningRequest) []byte {
	t.Helper()

	body, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	return body
}
