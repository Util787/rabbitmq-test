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

	// making move channel
	moveQueueName := routing.ArmyMovesPrefix + "." + username
	moveRoutingKey := routing.ArmyMovesPrefix + ".*"

	pubMoveChan, err := RMQConnection.Channel()
	if err != nil {
		log.Println("Failed to create channel on client: ", err)
	}

	// war channel for move handler
	warChan, _, err := pubsub.DeclareAndBind(RMQConnection, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, routing.WarRecognitionsPrefix+".*", pubsub.DurableQueue)
	if err != nil {
		log.Println("Fail to declare and bind war queue: ", err)
	}

	// move handler
	_, _, err = pubsub.SubscribeJSON(RMQConnection, perilTopicExchange, moveQueueName, moveRoutingKey, pubsub.TransientQueue, handlerMove(gameState, warChan, username))
	if err != nil {
		log.Println("Failed to subscribe: ", err)
	}

	// war handler
	_, _, err = pubsub.SubscribeJSON(RMQConnection, perilTopicExchange, routing.WarRecognitionsPrefix, routing.WarRecognitionsPrefix+".*", pubsub.DurableQueue, handlerWar(gameState, RMQConnection))
	if err != nil {
		log.Println("Failed to subscribe: ", err)
	}

	// command processing loop
	//---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
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
			err = pubsub.PublishJSON(pubMoveChan, perilTopicExchange, moveRoutingKey, move)
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Println("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

// accepts a war channel
func handlerMove(gs *gamelogic.GameState, warChan *amqp.Channel, username string) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Println("> ")

		makeWarRoutingKey := routing.WarRecognitionsPrefix + "." + username
		var ackType pubsub.AckType

		moveOutcome := gs.HandleMove(move)
		switch moveOutcome {
		case gamelogic.MoveOutComeSafe:
			ackType = pubsub.Ack

		case gamelogic.MoveOutcomeMakeWar:
			ackType = pubsub.Ack

			err := pubsub.PublishJSON(warChan, routing.ExchangePerilTopic, makeWarRoutingKey, gamelogic.RecognitionOfWar{Attacker: move.Player, Defender: gs.Player})
			if err != nil {
				log.Println("Error during MoveOutcomeMakeWar in move handler: ", err)
				ackType = pubsub.NackRequeue
			}

		case gamelogic.MoveOutcomeSamePlayer:
			ackType = pubsub.NackDiscard

		default:
			ackType = pubsub.NackDiscard
		}
		return ackType
	}
}

func handlerWar(gs *gamelogic.GameState, conn *amqp.Connection) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(rw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Println("> ")

		var ackType pubsub.AckType
		var message string

		outcome, winner, loser := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			ackType = pubsub.NackRequeue

		case gamelogic.WarOutcomeNoUnits:
			ackType = pubsub.NackDiscard

		case gamelogic.WarOutcomeOpponentWon:
			ackType = pubsub.Ack
			message = fmt.Sprintf("%v won a war against %v", winner, loser)

		case gamelogic.WarOutcomeYouWon:
			ackType = pubsub.Ack
			message = fmt.Sprintf("%v won a war against %v", winner, loser)

		case gamelogic.WarOutcomeDraw:
			ackType = pubsub.Ack
			message = fmt.Sprintf("A war between %v and %v resulted in a draw", winner, loser)

		default:
			log.Println("Error")
			ackType = pubsub.NackDiscard
		}

		routingKey := routing.GameLogSlug + "." + rw.Attacker.Username
		logChan, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routingKey, pubsub.DurableQueue)
		if err != nil {
			log.Println("Fail to declare and bind war queue: ", err)
			return pubsub.NackRequeue
		}

		err = pubsub.PublishGob(logChan, routing.ExchangePerilTopic, routingKey, routing.GameLog{CurrentTime: time.Now(), Message: message})
		if err != nil {
			log.Println("Failed to publish gob: ", err)
			return pubsub.NackRequeue
		}

		return ackType
	}
}
