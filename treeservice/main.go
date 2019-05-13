package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/tree"
	"github.com/urfave/cli"
)

type treeServiceActor struct {
	tokens    map[int64]string
	trees     map[int64]*actor.PID
	idCounter int64
}

func (state *treeServiceActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *messages.CreateTreeRequest:
		id := state.idCounter
		state.idCounter++
		token := make([]byte, 4)
		_, _ = rand.Read(token)

		state.tokens[id] = fmt.Sprintf("%x", token)
		state.trees[id] = context.Spawn(actor.PropsFromProducer(tree.NodeActorProducer))

		log.Printf("Treeservice creates tree with id %d", id)
		context.Send(state.trees[id], &messages.CreateTreeRequest{MaxSize: msg.MaxSize})
		context.Respond(
			&messages.CreateTreeResponse{Credentials: &messages.Credentials{Id: id, Token: state.tokens[id]}},
		)
	case *messages.SearchRequest:
		if _, exists := state.trees[msg.Credentials.Id]; !exists {
			log.Printf("No such tree with id %d", msg.Credentials.Id)
			context.Respond(&messages.NoSuchTreeError{Id: msg.Credentials.Id})
			return
		}
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			log.Printf("Invalid credentials... treeservice denies access")
			context.Respond(&messages.InvalidTokenError{Credentials: msg.Credentials})
		} else {
			log.Printf(
				"Valid credentials... treeservice forwards searchrequest to %s",
				state.trees[msg.Credentials.Id].Id,
			)
			context.Forward(state.trees[msg.Credentials.Id])
		}
	case *messages.DeleteRequest:
		if _, exists := state.trees[msg.Credentials.Id]; !exists {
			log.Printf("No such tree with id %d", msg.Credentials.Id)
			context.Respond(&messages.NoSuchTreeError{Id: msg.Credentials.Id})
			return
		}
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			log.Printf("Invalid credentials... treeservice denies access")
			context.Respond(&messages.InvalidTokenError{Credentials: msg.Credentials})
		} else {
			log.Printf(
				"Valid credentials... treeservice forwards deleterequest to %s",
				state.trees[msg.Credentials.Id].Id,
			)
			context.Forward(state.trees[msg.Credentials.Id])
		}
	case *messages.InsertRequest:
		if _, exists := state.trees[msg.Credentials.Id]; !exists {
			log.Printf("No such tree with id %d", msg.Credentials.Id)
			context.Respond(&messages.NoSuchTreeError{Id: msg.Credentials.Id})
			return
		}
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			log.Printf("Invalid credentials... treeservice denies access")
			context.Respond(&messages.InvalidTokenError{Credentials: msg.Credentials})
		} else {
			log.Printf(
				"Valid credentials... treeservice forwards insertrequest to %s",
				state.trees[msg.Credentials.Id].Id,
			)
			context.Forward(state.trees[msg.Credentials.Id])
		}
	case *messages.TraverseRequest:
		if _, exists := state.trees[msg.Credentials.Id]; !exists {
			log.Printf("No such tree with id %d", msg.Credentials.Id)
			context.Respond(&messages.NoSuchTreeError{Id: msg.Credentials.Id})
			return
		}
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			log.Printf("Invalid credentials... treeservice denies access.")
			context.Respond(&messages.InvalidTokenError{Credentials: msg.Credentials})
		} else {
			log.Printf(
				"Valid credentials... treeservice forwards traverserequest to %s",
				state.trees[msg.Credentials.Id].Id,
			)
			context.Forward(state.trees[msg.Credentials.Id])
		}
	case *messages.DeleteTreeRequest:
		if _, exists := state.trees[msg.Credentials.Id]; !exists {
			log.Printf("No such tree with id %d", msg.Credentials.Id)
			context.Respond(&messages.NoSuchTreeError{Id: msg.Credentials.Id})
			return
		}
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			log.Printf("Invalid credentials... treeservice denies access")
			context.Respond(&messages.InvalidTokenError{Credentials: msg.Credentials})
		} else {
			log.Printf("Valid credentials... Poisoning tree %d and deleting its data", msg.Credentials.Id)
			context.Poison(state.trees[msg.Credentials.Id])
			delete(state.trees, msg.Credentials.Id)
			delete(state.tokens, msg.Credentials.Id)
			context.Respond(&messages.DeleteTreeResponse{Credentials: msg.Credentials})
		}
	}
}

func newTreeServiceActor() actor.Actor {
	myActor := treeServiceActor{}
	myActor.idCounter = 1
	myActor.tokens = make(map[int64]string)
	myActor.trees = make(map[int64]*actor.PID)
	return &myActor
}

func main() {
	app := cli.NewApp()

	app.Author = "Dimitri Krivoj"
	app.Email = "krivoj@hm.edu"
	app.Version = "1.0.0"
	app.Name = "treeservice"
	app.Usage = "proto.actor service for managing search trees"
	app.UsageText = "treeservice [global options] command [arguments...]"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bind",
			Usage: "the treeservice will listen on this address",
			Value: "localhost:8090",
		},
	}
	app.Action = func(c *cli.Context) error {
		var wg sync.WaitGroup
		wg.Add(1)
		remote.Register("treeservice", actor.PropsFromProducer(newTreeServiceActor))
		remote.Start(c.String("bind"))
		wg.Wait()
		return nil
	}
	_ = app.Run(os.Args)
}
