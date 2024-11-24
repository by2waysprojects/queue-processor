package ws

import (
	"github.com/gorilla/websocket"
)

func HandleWSClient(connection *websocket.Conn, wsMessage chan *ActionClient) {
	for {
		var message ActionClient

		if err := connection.ReadJSON(&message); err != nil {
			close(wsMessage)
			break
		}
		wsMessage <- &message
	}
}
