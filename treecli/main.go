package main

import (
	"fmt"
	"os"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
	"github.com/urfave/cli"
)

func main() {
	timeout := 5 * time.Second
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bind",
			Usage: "address treecli should use",
			Value: "treecli.actors:8091",
		},
		cli.StringFlag{
			Name:  "remote",
			Usage: "address of the treeservice",
			Value: "treeservice.actors:8090",
		},
		cli.Int64Flag{
			Name:  "id",
			Usage: "id of the tree you want to alter",
		},
		cli.StringFlag{
			Name:  "token",
			Usage: "token to authorize your access for the specified tree",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "createtree",
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name:  "maxsize",
					Usage: "max size of a leaf",
					Value: 2,
				},
			},
			Action: func(c *cli.Context) error {
				remote.Start(c.GlobalString("bind"))
				pidResp, err := remote.SpawnNamed(
					c.GlobalString("remote"),
					"remote",
					"treeservice",
					timeout,
				)
				if err != nil {
					panic(err)
				}
				pid := pidResp.Pid
				res, err := actor.EmptyRootContext.RequestFuture(
					pid,
					&messages.CreateTreeRequest{MaxSize: c.Int64("maxsize")},
					timeout,
				).Result()
				if err != nil {
					panic(err)
				}
				response := res.(*messages.CreateTreeResponse)
				fmt.Printf("id: %d, token: %s\n", response.Credentials.Id, response.Credentials.Token)
				return nil
			},
		},
		{
			Name: "insert",
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name: "key",
				},
				cli.StringFlag{
					Name: "value",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.IsSet("key") || !c.IsSet("value") {
					panic("Missing key or value.")
				}
				if !c.GlobalIsSet("id") || !c.GlobalIsSet("token") {
					panic("Missing credentials.")
				}
				remote.Start(c.GlobalString("bind"))
				pidResp, err := remote.SpawnNamed(
					c.GlobalString("remote"),
					"remote",
					"treeservice",
					timeout,
				)
				if err != nil {
					panic(err)
				}
				pid := pidResp.Pid
				res, err := actor.EmptyRootContext.RequestFuture(
					pid,
					&messages.InsertRequest{
						Credentials: &messages.Credentials{
							Token: c.GlobalString("token"),
							Id:    c.GlobalInt64("id"),
						},
						Key:   c.Int64("key"),
						Value: c.String("value"),
					},
					timeout,
				).Result()
				if err != nil {
					panic(err)
				}
				response := res.(*messages.InsertResponse)
				switch response.Type {
				case messages.SUCCESS:
					fmt.Printf("(%d, %s) successfully inserted\n", c.Int64("key"), c.String("value"))
				case messages.KEY_ALREADY_EXISTS:
					panic(fmt.Sprintf("Tree already contains key %d", c.Int64("key")))
				case messages.ACCESS_DENIED:
					panic("Invalid credentials")
				case messages.NO_SUCH_TREE:
					panic("No such tree")
				default:
					panic("Unknown response type")
				}
				return nil
			},
		},
	}
	_ = app.Run(os.Args)
}
