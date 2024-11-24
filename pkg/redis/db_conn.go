package redis

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/redis/go-redis/v9"
)

type DbConn struct {
	db *redis.Client
}

func NewDbConn(address string) *DbConn {
	return &DbConn{
		db: redis.NewClient(&redis.Options{Addr: address}),
	}
}

func (dbConn *DbConn) Publish(ctx context.Context, key string, message []byte) error {
	return dbConn.db.Publish(ctx, key, message).Err()
}

func (dbConn *DbConn) Subscribe(ctx context.Context, key string) <-chan *redis.Message {
	return dbConn.db.Subscribe(ctx, key).Channel()
}

func (dbConn *DbConn) List(ctx context.Context, key string) ([]string, error) {
	return dbConn.db.LRange(ctx, key, 0, -1).Result()
}

func (dbConn *DbConn) Remove(ctx context.Context, list string, key interface{}) error {
	return dbConn.db.LRem(ctx, list, 1, key).Err()
}

func (dbConn *DbConn) PublishAndPush(ctx context.Context, queue string, list string, client []byte) error {
	luaScript := `
			local msg = ARGV[1]
			local channel = KEYS[1]
			local list = KEYS[2]
			redis.call('PUBLISH', channel, msg)
			redis.call('LPUSH', list, msg)
			return msg
		`
	_, err := dbConn.db.Eval(ctx, luaScript, []string{queue, list}, client).Result()
	return err
}

func (dbConn *DbConn) Update(ctx context.Context, key string, list string, client []byte) error {
	listClients, err := dbConn.List(ctx, list)
	if err != nil {
		return err
	}

	index := slices.IndexFunc(listClients, func(clientFromList string) bool {
		var clientParsed map[string]interface{}
		json.Unmarshal([]byte(clientFromList), clientParsed)
		for keyClient, _ := range clientParsed {
			return keyClient == key
		}
		return false
	})

	if index == -1 {
		return errors.New("Item not found")
	}

	return dbConn.db.LSet(ctx, key, int64(index), client).Err()
}

func (dbConn *DbConn) Get(ctx context.Context, list string, id string) ([]byte, error) {
	listClients, err := dbConn.List(ctx, list)
	if err != nil {
		return nil, err
	}

	index := slices.IndexFunc(listClients, func(clientFromList string) bool {
		var clientParsed map[string]interface{}
		json.Unmarshal([]byte(clientFromList), clientParsed)
		for keyClient, _ := range clientParsed {
			return keyClient == id
		}
		return false
	})

	if index == -1 {
		return nil, errors.New("Item not found")
	}

	return []byte(listClients[index]), nil
}
