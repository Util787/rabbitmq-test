package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")

	connectionString := "amqp://guest:guest@localhost:5672/"

	// exchanges
	perilDirectExchange := routing.ExchangePerilDirect
	perilTopicExchange := routing.ExchangePerilTopic

	RMQConnection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal("Failed to create connection with AMPQ URI on client: ", err)
	}
	defer RMQConnection.Close()
	log.Println("AMQP URI connected successfully on client")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal("Failed to get username on client:", err)
	}
	gameState := gamelogic.NewGameState(username)

	// making a pause queue and subscribing
	pauseQueueName := routing.PauseKey + "." + username

	_, _, err = pubsub.SubscribeJSON(RMQConnection, perilDirectExchange, pauseQueueName, routing.PauseKey, pubsub.TransientQueue, handlerPause(gameState))
	if err != nil {
		log.Println("Failed to subscribe: ", err)
	}

	// making a move queue and subscribing
	moveQueueName := routing.ArmyMovesPrefix + "." + username
	moveRoutingKey := routing.ArmyMovesPrefix + ".*"
	
	moveChan, _, err := pubsub.SubscribeJSON(RMQConnection, perilTopicExchange, moveQueueName, moveRoutingKey, pubsub.TransientQueue, handlerMove(gameState))
	if err != nil {
		log.Println("Failed to subscribe: ", err)
	}

	// command processing loop
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "spawn":
			if len(input) < 3 {
				log.Println("Wrong syntax, usage: spawn <location> <rank>")
				continue
			}
			_, exist := gamelogic.PossibleLocations[input[1]]
			_, exist1 := gamelogic.PossibleUnits[input[2]]
			if !(exist && exist1) {
				fmt.Println("Unknown location or rank, Here are allowed locations and ranks")
				fmt.Println("Locations:")
				for location := range gamelogic.PossibleLocations { // using "for" here may be unrational but I considered that the game logic could change
					fmt.Println("- ", location)
				}
				fmt.Println("Units:")
				for unit := range gamelogic.PossibleUnits {
					fmt.Println("- ", unit)
				}
				continue
			}
			err = gameState.CommandSpawn(input)
			if err != nil {
				log.Println("Failed to execute spawn command: ", err)
			}
		case "move":
			move, err := gameState.CommandMove(input)
			if err != nil {
				log.Println("Failed to execute move command: ", err)
				continue
			}
			err = pubsub.PublishJSON(moveChan, perilTopicExchange, moveRoutingKey, move)
			if err != nil {
				log.Println("Failed to publish the move")
				continue
			}
			fmt.Printf("User: %v moved units:%v to location: %v\n", username, move.Units, move.ToLocation)
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(move)
	}
}
