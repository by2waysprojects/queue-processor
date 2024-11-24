package ws

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type Client struct {
	ClientID      string
	WebsocketConn *websocket.Conn
	HearthBeat    time.Time
}

func NewClient(clientID string, webSocketConn *websocket.Conn, hearthbeat time.Time) *Client {
	return &Client{
		ClientID:      clientID,
		WebsocketConn: webSocketConn,
		HearthBeat:    hearthbeat,
	}
}

func (client *Client) SellTicket(jwtKey string, ctx context.Context) error {

	tokenString, err := generateJWT(client.ClientID, jwtKey)
	if err != nil {
		return err
	}

	statusClient := NewStatusClient(client.ClientID, true, 0, 0, tokenString)
	if err := client.WebsocketConn.WriteJSON(&statusClient); err != nil {
		return err
	}

	return nil

}

func (client *Client) SendStatus(time, position int) error {
	statusClient := NewStatusClient(client.ClientID, false, time, position, "")
	if err := client.WebsocketConn.WriteJSON(&statusClient); err != nil {
		return err
	}
	return nil
}

func generateJWT(userID string, jwtKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(time.Hour * 1).Unix(),
	})
	tokenString, err := token.SignedString([]byte(jwtKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
