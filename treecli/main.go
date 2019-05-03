package main

import (
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
	"github.com/urfave/cli"
	"os"
	"time"
)

func main() {
	timeout := 50 * time.Millisecond
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bind",
			Usage: "the address treecli shall bind to",
			Value: "treecli.actors:8091",
		},
		cli.StringFlag{
			Name:  "remote",
			Usage: "the address treeservice is bound to",
			Value: "treeservice.actors:8090",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "createtree",
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name:  "maxsize",
					Usage: "maximal size of a leaf",
					Value: 2,
				},
			},
			Action: func(c *cli.Context) error {
				remote.Start(c.String("bind"))
				pidResp, _ := remote.Spawn(c.String("remote"), "treeservice", timeout)
				pid := pidResp.Pid
				res, _ := actor.EmptyRootContext.RequestFuture(
					pid,
					messages.CreateTreeRequest{
						MaxSize: c.Int64("maxsize")},
					timeout,
				).Result()
				response := res.(messages.CreateTreeResponse)
				fmt.Printf("%d, %s", response.Credentials.Id, response.Credentials.Token)
				return nil
			},
		},
	}
	_ = app.Run(os.Args)
}
