package domain

type Request interface {
	CleaningRequest | WashRequest | RestockRequest
}

type CleaningRequest struct {
	RoomName string
	R        string
}

type WashRequest struct {
	RoomName string
	Linens   bool
	Towels   bool
}

type RestockRequest struct {
	ItemsName []string
}
