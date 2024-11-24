package ws

type StatusClient struct {
	ClientID     string `json:"client_id"`
	Position     int    `json:"position"`
	IsReadyToBuy bool   `json:"is_ready_to_buy"`
	TimeLeft     int    `json:"time_left"`
	TokenToBuy   string `json:"token_to_buy"`
}

func NewStatusClient(clientID string, isReadyToBuy bool, timeLeft int, position int, tokenToBuy string) *StatusClient {
	return &StatusClient{
		ClientID:     clientID,
		IsReadyToBuy: isReadyToBuy,
		TimeLeft:     timeLeft,
		Position:     position,
		TokenToBuy:   tokenToBuy,
	}
}
