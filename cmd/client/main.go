package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connectionString := "amqp://guest:guest@localhost:5672/"

	exchange := routing.ExchangePerilDirect

	RMQConnection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal("Failed to create connection with AMPQ URI on client")
	}
	defer RMQConnection.Close()
	log.Println("AMQP URI connected successfully on client")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal("Failed to get username on client:", err)
	}

	queueName := routing.PauseKey + "." + username

	subChan, subQueue, err := pubsub.DeclareAndBind(RMQConnection, exchange, queueName, routing.PauseKey, pubsub.TransientQueue)
	if err != nil {
		log.Fatal("Failed to declare and bind on client")
	}
	fmt.Println("Created: ", subChan, subQueue)

	time.Sleep(5 * time.Minute)
}
