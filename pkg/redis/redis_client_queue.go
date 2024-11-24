package redis

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"
)

type RedisClientQueue struct {
	ClientID   string    `json:"client_id"`
	HearthBeat time.Time `json:"hearth_beat"`
	TokenToBuy string    `json:"token_to_buy"`
	Birthday   time.Time `json:"birthday"`
}

func NewRedisClientQueue(clientID string, hearthbeat time.Time, birthday time.Time) *RedisClientQueue {
	return &RedisClientQueue{
		ClientID:   clientID,
		HearthBeat: hearthbeat,
		Birthday:   birthday,
	}
}

func AddClientToClientQueueRedis(clientID string, ctx context.Context, redisClient *DbConn, creationTime time.Time, queueName, listName string) error {
	clientSerialized, err := SerializeRedisClientQueue(NewRedisClientQueue(clientID, creationTime, creationTime))
	if err != nil {
		return err
	}

	err = redisClient.PublishAndPush(ctx, queueName, listName, clientSerialized)
	if err != nil {
		return err
	}

	return nil
}

func CalculateTimeLeftAndPosition(ClientID string, ctx context.Context, redisClient *DbConn, clientsPerGroup int, timeBuying int, redisClientQueue []*RedisClientQueue) (int, int) {
	index := slices.IndexFunc(redisClientQueue, func(clientFromList *RedisClientQueue) bool {
		return clientFromList.ClientID == ClientID
	})

	if index == -1 {
		return 0, 0
	}
	groupsBefore := (index + clientsPerGroup) / clientsPerGroup
	return groupsBefore * timeBuying, index + 1
}

func RemoveIfExistsClientFromRedis(clientID *RedisClientQueue, ctx context.Context, redisClient *DbConn, listName string) error {
	clientRedisSerialized, err := SerializeRedisClientQueue(clientID)
	if err != nil {
		return err
	}
	err = redisClient.Remove(ctx, listName, clientRedisSerialized)
	if err != nil {
		return err
	}
	return nil
}

func UpdateClientFromRedis(redisClientQueue *RedisClientQueue, ctx context.Context, redisClient *DbConn, listName string, key string) error {
	client, err := SerializeRedisClientQueue(redisClientQueue)
	if err != nil {
		return err
	}

	err = redisClient.Update(ctx, key, listName, client)
	if err != nil {
		return err
	}
	return nil
}

func ListRedisClientQueue(ctx context.Context, redisClient *DbConn, listName string) (error, []*RedisClientQueue) {
	redisClientListasString, err := redisClient.List(ctx, listName)
	if err != nil {
		return errors.New("Error obtaining message history"), nil
	}

	redisClientList := make([]*RedisClientQueue, 0)

	for _, clientString := range redisClientListasString {
		clientRedis, err := DeserializeRedisClientQueue([]byte(clientString))
		if err != nil {
			return err, nil
		}
		redisClientList = append(redisClientList, clientRedis)
	}
	return nil, redisClientList
}

func DeserializeRedisClientQueue(clientSerialized []byte) (*RedisClientQueue, error) {
	var deserializedClient map[string]RedisClientQueue
	err := json.Unmarshal(clientSerialized, &deserializedClient)
	if err != nil {
		return nil, err
	}
	for _, v := range deserializedClient {
		return &v, nil
	}
	return nil, nil
}

func SerializeRedisClientQueue(client *RedisClientQueue) ([]byte, error) {
	var serializedClient map[string]interface{}
	serializedClient[client.ClientID] = client
	return json.Marshal(serializedClient)
}

func FindByClientID(clientID string, redisClient *DbConn, listName string, ctx context.Context) (*RedisClientQueue, error) {
	clientSerialized, err := redisClient.Get(ctx, listName, clientID)
	if err != nil {
		return nil, err
	}

	return DeserializeRedisClientQueue(clientSerialized)
}
