package ticketmanager

import (
	"context"
	"queue-processor/pkg/redis"
	"queue-processor/pkg/ws"
	"sync"
	"time"

	"go.uber.org/zap"
)

type TicketManager struct {
	ClientQueueArray      []*ws.Client
	FinishedClient        chan *ws.Client
	JwtKey                string
	ClientsPerGroup       int
	ClientQueueArrayMutex sync.Mutex
	Ctx                   context.Context
	RedisClient           *redis.DbConn
	TimeBuying            int
	Logger                *zap.Logger
	MaxTimeAlive          time.Duration
}

func NewTicketManager(maxClients int, queueSize int,
	jwtKey string, redisAddress string, timeBuying int, logger *zap.Logger, maxTimeAlive int) *TicketManager {

	redisClient := redis.NewDbConn(redisAddress)

	ctx := context.Background()

	finishedClientPool := make(chan *ws.Client, queueSize)
	clientQueueArray := make([]*ws.Client, 0)

	return &TicketManager{
		JwtKey:           jwtKey,
		ClientQueueArray: clientQueueArray,
		FinishedClient:   finishedClientPool,
		ClientsPerGroup:  maxClients,
		RedisClient:      redisClient,
		Ctx:              ctx,
		TimeBuying:       timeBuying,
		Logger:           logger,
		MaxTimeAlive:     time.Duration(maxTimeAlive) * time.Minute,
	}
}

func (tm *TicketManager) FinishClients() {
	for {
		finishedClient := <-tm.FinishedClient

		tm.Logger.Debug("tm - closing connection", zap.String("clientID", finishedClient.ClientID))

		if removed := tm.RemoveIfExistsClient(finishedClient); !removed {
			tm.Logger.Debug("tm - client not found in array", zap.String("clientID", finishedClient.ClientID))
		}

		if err := finishedClient.WebsocketConn.Close(); err != nil {
			tm.Logger.Error("tm - connection already closed", zap.Error(err))
		}
	}
}

func (tm *TicketManager) SendPeriodicStatus() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		<-ticker.C

		tm.Logger.Debug("tm - sending periodic status")

		err, listRedisClientQueue := redis.ListRedisClientQueue(tm.Ctx, tm.RedisClient, "messageHistory")
		if err != nil {
			tm.Logger.Error("tm - error obtaining message history", zap.Error(err))
			break
		}

		for _, client := range tm.ClientQueueArray {
			time, position := redis.CalculateTimeLeftAndPosition(client.ClientID, tm.Ctx, tm.RedisClient, tm.ClientsPerGroup, tm.TimeBuying, listRedisClientQueue)
			if err := client.SendStatus(time, position); err != nil {
				tm.Logger.Error("tm - error sending status", zap.Error(err))
			}
		}
	}
}
