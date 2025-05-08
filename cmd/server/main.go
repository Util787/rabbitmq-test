package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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

	go sigIntHandle(RMQConnection)

	pubChan, err := RMQConnection.Channel()
	if err != nil {
		log.Fatal("Failed to create RMQ channel on server")
	}

	err = PublishPauseMessage(pubChan, exchange, routing.PauseKey)
	if err != nil {
		log.Println("Failed to publish message on server")
	}

	gamelogic.PrintServerHelp()
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "pause":
			err = PublishPauseMessage(pubChan, exchange, routing.PauseKey)
			if err != nil {
				log.Println("Failed to publish message on server")
			}
		case "resume":
			err = PublishResumeMessage(pubChan, exchange, routing.PauseKey)
			if err != nil {
				log.Println("Failed to publish message on server")
			}
		case "quit":
			log.Println("Quitting")
			RMQConnection.Close()
			os.Exit(1)
		default:
			log.Println("Unknown command")
		}

	}

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
}

func PublishPauseMessage(ch *amqp.Channel, exchange, key string) error {
	log.Println("Sending pause message")

	playState := routing.PlayingState{
		IsPaused: true,
	}

	err := pubsub.PublishJSON(ch, exchange, routing.PauseKey, playState)
	if err != nil {
		return err
	}
	return nil
}

func PublishResumeMessage(ch *amqp.Channel, exchange, key string) error {
	log.Println("Sending resume message")

	playState := routing.PlayingState{
		IsPaused: false,
	}

	err := pubsub.PublishJSON(ch, exchange, routing.PauseKey, playState)
	if err != nil {
		return err
	}
	return nil
}
