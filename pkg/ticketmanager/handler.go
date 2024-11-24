package ticketmanager

import (
	"fmt"
	"net/http"
	"queue-processor/pkg/redis"
	"queue-processor/pkg/ws"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	FINISHED_STATUS    = "finished"
	HEARTH_BEAT_STATUS = "alive"
	RECONNECT_STATUS   = "reconnect"
	NEW_CLIENT_STATUS  = "newclient"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func (tm *TicketManager) HandleTicketSale(w http.ResponseWriter, r *http.Request) {
	creationTime := time.Now()
	chanFinisher := make(chan string)
	chanSeller := make(chan string)
	chanWS := make(chan *ws.ActionClient)
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		tm.Logger.Error("tm - error upgrading connection", zap.Error(err))
		return
	}

	clientID := fmt.Sprintf("%d", creationTime.UnixNano())
	client := ws.NewClient(clientID, conn, creationTime)

	tm.Logger.Debug("tm - new client", zap.String("clientID", clientID))

	go ws.HandleWSClient(conn, chanWS)

	for {
		select {
		case message, open := <-chanWS:
			tm.Logger.Debug("tm - new message", zap.String("action", message.Action), zap.String("clientID", message.ClientID))
			if !open {
				tm.RemoveIfExistsClient(client)
				return
			}
			client.HearthBeat = time.Now()

			switch message.Action {
			case FINISHED_STATUS:
				if err := tm.sendFinishedMessage(client.ClientID); err != nil {
					tm.Logger.Error("tm - error sending finished message", zap.Error(err))
				}
				tm.FinishedClient <- client
				return

			case HEARTH_BEAT_STATUS:
				updatedRedisClient := redis.NewRedisClientQueue(client.ClientID, client.HearthBeat, creationTime)
				redis.UpdateClientFromRedis(updatedRedisClient, tm.Ctx, tm.RedisClient, "messageHistory", client.ClientID)

			case NEW_CLIENT_STATUS:
				if err := tm.createNewClientAndSubscribe(chanSeller, chanFinisher, client); err != nil {
					tm.FinishedClient <- client
					return
				}

			case RECONNECT_STATUS:
				clientRedis, err := redis.FindByClientID(message.ClientID, tm.RedisClient, "messageHistory", tm.Ctx)

				if err != nil || clientRedis == nil || time.Since(clientRedis.HearthBeat) > tm.MaxTimeAlive {
					tm.Logger.Error("tm - error finding client by ID", zap.Error(err))
					tm.FinishedClient <- client
					return
				}

				client.ClientID = clientRedis.ClientID
				updatedRedisClient := redis.NewRedisClientQueue(client.ClientID, client.HearthBeat, clientRedis.Birthday)
				redis.UpdateClientFromRedis(updatedRedisClient, tm.Ctx, tm.RedisClient, "messageHistory", client.ClientID)
				tm.updateReconnectedClientAndSubscribe(chanSeller, chanFinisher, client)

			default:
			}
		case <-chanFinisher:

			tm.Logger.Debug("tm - received client by channel finisher", zap.String("clientID", clientID))

			_, index := tm.FindClientAndIndex(client)
			if index < 0 {
				tm.Logger.Debug("tm - client not found by ID", zap.String("clientID", clientID))
			}

			tm.FinishedClient <- client
			return

		case clientID := <-chanSeller:

			tm.Logger.Debug("tm - received client by channel finisher", zap.String("clientID", clientID))

			_, index := tm.FindClientAndIndex(client)
			if index < 0 {
				tm.Logger.Debug("tm - client not found by ID", zap.String("clientID", clientID))
				tm.FinishedClient <- client
				return
			}
			if err := client.SellTicket(tm.JwtKey, tm.Ctx); err != nil {
				tm.Logger.Error("tm - error selling ticket", zap.Error(err))
				tm.FinishedClient <- client
				return
			}
			if err := tm.sendSoldMessage(clientID); err != nil {
				tm.FinishedClient <- client
				return
			}
		}
	}
}

func (tm *TicketManager) HandleTicketFinished(w http.ResponseWriter, r *http.Request) {
	tm.Logger.Debug("tm - ticket finished handler")
	for _, client := range tm.ClientQueueArray {
		tm.FinishedClient <- client
	}
}

func (tm *TicketManager) sendSoldMessage(client string) error {
	err := redis.PublishMessageInRedis(tm.Ctx, tm.RedisClient, client, "sell-worker-"+client)
	if err != nil {
		tm.Logger.Error("tm - error publishing sold message", zap.Error(err))
		return err
	}

	return nil
}

func (tm *TicketManager) sendFinishedMessage(client string) error {
	err := redis.PublishMessageInRedis(tm.Ctx, tm.RedisClient, client, "finish-worker-"+client)
	if err != nil {
		tm.Logger.Error("tm - error publishing finished message", zap.Error(err))
		return err
	}

	return nil
}

func (tm *TicketManager) createNewClientAndSubscribe(chanSeller chan string, chanFinisher chan string, client *ws.Client) error {
	tm.AddClientToClientQueueArray(client)
	if err := redis.AddClientToClientQueueRedis(client.ClientID, tm.Ctx, tm.RedisClient, client.HearthBeat, "client-queue", "messageHistory"); err != nil {
		tm.Logger.Error("tm - error adding client to redis queue", zap.Error(err))
		return err
	}

	go redis.SubscribeRedis(tm.RedisClient, tm.Ctx, chanSeller, "sell-tm-"+client.ClientID)
	go redis.SubscribeRedis(tm.RedisClient, tm.Ctx, chanFinisher, "finish-tm-"+client.ClientID)

	return nil
}

func (tm *TicketManager) updateReconnectedClientAndSubscribe(chanSeller chan string, chanFinisher chan string, client *ws.Client) {
	tm.AddClientToClientQueueArray(client)
	go redis.SubscribeRedis(tm.RedisClient, tm.Ctx, chanSeller, "sell-tm-"+client.ClientID)
	go redis.SubscribeRedis(tm.RedisClient, tm.Ctx, chanFinisher, "finish-tm-"+client.ClientID)
}
