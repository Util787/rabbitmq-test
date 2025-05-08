package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")

	connectionString := "amqp://guest:guest@localhost:5672/"

	exchange := routing.ExchangePerilDirect

	RMQConnection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal("Failed to create connection with AMPQ URI on server")
	}
	defer RMQConnection.Close()
	log.Println("AMQP URI connected successfully on server")

	pubChan, err := RMQConnection.Channel()
	if err != nil {
		log.Fatal("Failed to create RMQ channel on server")
	}

	playState := routing.PlayingState{
		IsPaused: true,
	}

	err = pubsub.PublishJSON(pubChan, exchange, routing.PauseKey, playState)
	if err != nil {
		log.Println("Failed to publish message on server")
	}

	sigIntHandle(RMQConnection)
}

func sigIntHandle(RMQConnection *amqp.Connection) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	for sig := range signalChan {
		log.Println("Received signal:", sig)
		log.Println("Shutting down the programm")
		RMQConnection.Close()
		os.Exit(1)
	}

	//maybe better write this logic in main:
	//make chan without buf
	//signalChan := make(chan os.Signal) and:
	//go func() {
	//     <-signalChan
	//     *some logic*
	//     os.Exit(1)
	// }()
}
