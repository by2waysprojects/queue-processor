package main

import (
	"net/http"
	"os"
	"queue-processor/pkg/ticketmanager"
	"queue-processor/pkg/worker"
	"strconv"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		return
	}

	// Acceder a las variables de entorno
	serverPort := os.Getenv("SERVER_PORT")
	maxWorkers, _ := strconv.Atoi(os.Getenv("MAX_WORKERS"))
	queueSize, _ := strconv.Atoi(os.Getenv("QUEUE_SIZE"))
	redisAddress := os.Getenv("REDIS_ADDRESS")
	jwtKey := os.Getenv("JWT_KEY")
	timeBuying, _ := strconv.Atoi(os.Getenv("TIME_BUYING"))
	maxTimeAlive, _ := strconv.Atoi(os.Getenv("MAX_TIME_ALIVE"))

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	tm := ticketmanager.NewTicketManager(maxWorkers, queueSize, jwtKey, redisAddress, timeBuying, logger, maxTimeAlive)

	for i := 0; i < maxWorkers; i++ {
		worker := worker.NewWorker(redisAddress, jwtKey, timeBuying, maxTimeAlive, logger)
		go worker.ProcessClientQueue()
	}

	go tm.SendPeriodicStatus()
	go tm.FinishClients()

	http.HandleFunc("/on-queue", tm.HandleTicketSale)
	http.HandleFunc("/finish-queue", tm.HandleTicketFinished)
	http.HandleFunc("/reconnect", tm.HandleTicketSale)

	logger.Info("Starting server at port: ", zap.String("port", serverPort))
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		logger.Error("Error starting server: ", zap.Error(err))
	}
}
