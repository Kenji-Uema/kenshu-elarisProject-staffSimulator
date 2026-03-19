package domain

type Request interface {
	CleaningRequest | WashRequest | RestockRequest
}

type CleaningRequest struct {
	RoomName    string
	RequestType string
}

type WashRequest struct {
	RoomName string
	Item     string
}

type RestockRequest struct {
	ItemsName []string
}
