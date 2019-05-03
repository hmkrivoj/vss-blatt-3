package main

import (
	"crypto/rand"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/tree"
	"github.com/urfave/cli"
	"os"
	"sync"
)

var wg = &sync.WaitGroup{}

type treeServiceActor struct {
	tokens    map[int64]string
	trees     map[int64]*actor.PID
	idCounter int64
}

func (state *treeServiceActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case actor.PoisonPill:
		for _, pid := range state.trees {
			context.Poison(pid)
		}
		wg.Done()
	case messages.CreateTreeRequest:
		id := state.idCounter
		state.idCounter++
		token := make([]byte, 4)
		_, _ = rand.Read(token)

		state.tokens[id] = fmt.Sprintf("%x", token)
		state.trees[id] = context.Spawn(actor.PropsFromProducer(tree.NodeActorProducer))

		context.Send(state.trees[id], messages.CreateTreeRequest{MaxSize: msg.MaxSize})
		context.Respond(
			&messages.CreateTreeResponse{Credentials: &messages.Credentials{Id: id, Token: state.tokens[id]}},
		)
	case messages.SearchRequest:
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			context.Respond(&messages.SearchResponse{Key: msg.Key, Type: messages.ACCESS_DENIED})
		} else {
			context.Forward(state.trees[msg.Credentials.Id])
		}
	case messages.InsertRequest:
		if state.tokens[msg.Credentials.Id] != msg.Credentials.Token {
			context.Respond(&messages.InsertResponse{Key: msg.Key, Type: messages.ACCESS_DENIED})
		} else {
			context.Forward(state.trees[msg.Credentials.Id])
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
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bind",
			Usage: "the address treeservice shall bind to",
			Value: "treeservice.actors:8090",
		},
	}
	app.Action = func(c *cli.Context) error {
		wg.Add(1)
		remote.Register("treeservice", actor.PropsFromProducer(newTreeServiceActor))
		remote.Start(c.String("bind"))
		wg.Wait()
		return nil
	}
	_ = app.Run(os.Args)
}
