package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	connectionString := "amqp://guest:guest@localhost:5672/"

	RMQConnection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal("Failed to create connection with AMPQ URI")
	}
	defer RMQConnection.Close()
	log.Println("AMPQ URI connected successfully")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	sig := <-signalChan
	log.Println("Received signal:", sig)
	log.Println("Shutting down the programm")

	//better this logic:
	//make chan without buf
	//signalChan := make(chan os.Signal) and:
	//go func() {
	//     <-signalChan
	//     *some logic*
	//     os.Exit(1)
	// }()
}
