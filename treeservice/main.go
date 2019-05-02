package main

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/urfave/cli"
	"os"
)

type treeserviceActor struct {
	tokens map[int64]string
	trees  map[int64]actor.PID
}

func (*treeserviceActor) Receive(c actor.Context) {
	panic("implement me")
}

func newTreeserviceActor() actor.Actor {
	return &treeserviceActor{}
}

func init() {
	remote.Register("treeservice", actor.PropsFromProducer(newTreeserviceActor))
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
		remote.Start(c.String("bind"))
		return nil
	}
	_ = app.Run(os.Args)
}
