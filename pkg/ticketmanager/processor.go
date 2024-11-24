package ticketmanager

import (
	"queue-processor/pkg/ws"
	"slices"
)

func (tm *TicketManager) FindClientAndIndex(client *ws.Client) (*ws.Client, int) {
	index := slices.IndexFunc(tm.ClientQueueArray, func(clientToFind *ws.Client) bool {
		return clientToFind.ClientID == client.ClientID
	})
	if index == -1 {
		return nil, -1
	}
	return tm.ClientQueueArray[index], index
}

func (tm *TicketManager) RemoveIfExistsClient(client *ws.Client) bool {
	tm.ClientQueueArrayMutex.Lock()
	defer tm.ClientQueueArrayMutex.Unlock()
	_, index := tm.FindClientAndIndex(client)

	if index < 0 {
		return false
	}

	tm.ClientQueueArray = append(tm.ClientQueueArray[:index], tm.ClientQueueArray[index+1:]...)
	return true
}

func (tm *TicketManager) AddClientToClientQueueArray(client *ws.Client) {
	tm.ClientQueueArrayMutex.Lock()
	defer tm.ClientQueueArrayMutex.Unlock()
	tm.ClientQueueArray = append(tm.ClientQueueArray, client)
}
