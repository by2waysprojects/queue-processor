package worker

import (
	"context"
	"queue-processor/pkg/redis"
	"time"

	"go.uber.org/zap"
)

type Worker struct {
	RedisClient  *redis.DbConn
	Ctx          context.Context
	JwtKey       string
	TimeBuying   int
	MaxTimeAlive time.Duration
	Logger       *zap.Logger
}

func NewWorker(redisAddress string, jwtKey string, timeBuying int, maxTimeAlive int, logger *zap.Logger) *Worker {
	redisClient := redis.NewDbConn(redisAddress)
	ctx := context.Background()
	return &Worker{
		RedisClient:  redisClient,
		Ctx:          ctx,
		JwtKey:       jwtKey,
		TimeBuying:   timeBuying,
		MaxTimeAlive: time.Duration(maxTimeAlive) * time.Minute,
		Logger:       logger,
	}
}

func (worker *Worker) ProcessClientQueue() {
	chanQueue := make(chan string)
	go redis.SubscribeRedis(worker.RedisClient, worker.Ctx, chanQueue, "client-queue")

	for {
		chanFinished := make(chan string)
		chanSold := make(chan string)
		idClient := <-chanQueue

		worker.Logger.Debug("worker - process new client queue", zap.String("clientID", idClient))

		redisQueueClient, err := redis.FindByClientID(idClient, worker.RedisClient, "messageHistory", worker.Ctx)
		if err != nil {
			worker.Logger.Error("worker - error finding client", zap.Error(err))
			continue
		}

		timeLeft := time.Since(redisQueueClient.HearthBeat)

		if timeLeft > worker.MaxTimeAlive {
			worker.Logger.Debug("worker - client exceed max time alive", zap.String("clientID", idClient))

			if err := worker.sendFinishedMessage(redisQueueClient); err != nil {
				break
			}
			continue
		}

		go redis.SubscribeRedis(worker.RedisClient, worker.Ctx, chanFinished, "finish-worker-"+redisQueueClient.ClientID)
		go redis.SubscribeRedis(worker.RedisClient, worker.Ctx, chanSold, "sell-worker-"+redisQueueClient.ClientID)

		if err := worker.sendSoldMessage(redisQueueClient); err != nil {
			break
		}

		<-chanSold

		if err := redis.RemoveIfExistsClientFromRedis(redisQueueClient, worker.Ctx, worker.RedisClient, "messageHistory"); err != nil {
			worker.Logger.Error("worker - error removing client from redis", zap.Error(err))
			break
		}

		select {
		case <-chanFinished:
			worker.Logger.Debug("worker - received client finished", zap.String("clientID", idClient))
		case <-time.After(time.Duration(worker.TimeBuying) * time.Second):
			worker.Logger.Debug("worker - timeout client arrived", zap.String("clientID", idClient))
			if err := worker.sendFinishedMessage(redisQueueClient); err != nil {
				break
			}
		}
	}
}

func (worker *Worker) sendFinishedMessage(client *redis.RedisClientQueue) error {

	worker.Logger.Debug("worker - sending finish message", zap.String("clientID", client.ClientID))

	err := redis.PublishMessageInRedis(worker.Ctx, worker.RedisClient, client.ClientID, "finish-tm-"+client.ClientID)
	if err != nil {
		worker.Logger.Error("worker - error publishing finished message", zap.Error(err))
		return err
	}

	return nil
}

func (worker *Worker) sendSoldMessage(client *redis.RedisClientQueue) error {

	worker.Logger.Debug("worker - sending sell message", zap.String("clientID", client.ClientID))

	err := redis.PublishMessageInRedis(worker.Ctx, worker.RedisClient, client.ClientID, "sell-tm-"+client.ClientID)
	if err != nil {
		worker.Logger.Error("worker - error publishing message sold message", zap.Error(err))
		return err
	}

	return nil
}
