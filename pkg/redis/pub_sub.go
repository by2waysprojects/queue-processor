package redis

import (
	"context"
	"encoding/json"
)

func PublishMessageInRedis(ctx context.Context, redisClient *DbConn, client string, messageIdentifier string) error {
	clientJSON, err := json.Marshal(client)
	if err != nil {
		return err
	}

	err = redisClient.Publish(ctx, messageIdentifier, clientJSON)
	return err
}

func SubscribeRedis(redisClient *DbConn, ctx context.Context, channel chan string, subscriptionID string) error {
	subscriber := redisClient.Subscribe(ctx, subscriptionID)
	for {
		msg := <-subscriber
		var clientJson string
		err := json.Unmarshal([]byte(msg.Payload), clientJson)
		if err != nil {
			return err
		}
		channel <- clientJson
	}
}
