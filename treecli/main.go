package main

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/remote"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
	"github.com/urfave/cli"
)

const timeout = 60 * time.Second

const globalFlagID = "id"
const globalFlagToken = "token"

func handleCredentialsFromCliContext(c *cli.Context) {
	if !c.GlobalIsSet(globalFlagID) || !c.GlobalIsSet(globalFlagToken) {
		panic("Missing credentials.")
	}
}

func requestAndWait(context *actor.RootContext, wg *sync.WaitGroup, remotePid *actor.PID, pid *actor.PID, message interface{}) {
	wg.Add(1)
	context.RequestWithCustomSender(remotePid, message, pid)
	wg.Wait()
}

type treeCliActor struct {
	wg *sync.WaitGroup
}

func (state *treeCliActor) Receive(c actor.Context) {
	switch msg := c.Message().(type) {
	case *messages.NoSuchTreeError:
		c.Stop(c.Self())
		fmt.Printf("No tree with id %d", msg.Id)
	case *messages.NoSuchKeyError:
		c.Stop(c.Self())
		fmt.Printf("Tree contains no key %d", msg.Key)
	case *messages.InvalidTokenError:
		c.Stop(c.Self())
		fmt.Printf("Invalid token %s for tree %d", msg.Credentials.Token, msg.Credentials.Id)
	case *messages.KeyAlreadyExistsError:
		c.Stop(c.Self())
		fmt.Printf("Tree already contains item (%d, %s)", msg.Item.Key, msg.Item.Value)
	case *messages.CreateTreeResponse:
		c.Stop(c.Self())
		fmt.Printf("id: %d, token: %s\n", msg.Credentials.Id, msg.Credentials.Token)
	case *messages.InsertResponse:
		c.Stop(c.Self())
		fmt.Printf("(%d, %s) successfully inserted\n", msg.Item.Key, msg.Item.Value)
	case *messages.SearchResponse:
		c.Stop(c.Self())
		fmt.Printf("Found item (%d, %s)\n", msg.Item.Key, msg.Item.Value)
	case *messages.DeleteResponse:
		c.Stop(c.Self())
		fmt.Printf("Successfully deleted item (%d, %s) from tree\n", msg.Item.Key, msg.Item.Value)
	case *messages.TraverseResponse:
		for _, item := range msg.Items {
			fmt.Printf("(%d, %s), ", item.Key, item.Value)
		}
		c.Stop(c.Self())
		fmt.Println()
	case *actor.Stopped:
		state.wg.Done()
	}
}

func main() {
	var rootContext = actor.EmptyRootContext
	var wg sync.WaitGroup
	var bindAddr, remoteAddr string
	var pid, remotePid *actor.PID

	app := cli.NewApp()
	app.Author = "Dimitri Krivoj"
	app.Email = "krivoj@hm.edu"
	app.Version = "1.0.0"
	app.Name = "treecli"
	app.Usage = "communication with treeservice"
	app.UsageText = "treecli [global options] command [arguments...]"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "bind",
			Usage:       "address treecli should use",
			Value:       "treecli.actors:8091",
			Destination: &bindAddr,
		},
		cli.StringFlag{
			Name:        "remote",
			Usage:       "address of the treeservice",
			Value:       "treeservice.actors:8090",
			Destination: &remoteAddr,
		},
		cli.Int64Flag{
			Name:  globalFlagID,
			Usage: "id of the tree you want to alter",
		},
		cli.StringFlag{
			Name:  globalFlagToken,
			Usage: "token to authorize your access for the specified tree",
		},
	}

	app.Before = func(c *cli.Context) error {
		remote.Start(bindAddr)
		props := actor.PropsFromProducer(func() actor.Actor {
			myActor := treeCliActor{wg: &wg}
			return &myActor
		})
		pidResp, err := remote.SpawnNamed(
			remoteAddr,
			"remote",
			"treeservice",
			timeout,
		)
		if err == nil {
			remotePid = pidResp.Pid
			pid = rootContext.Spawn(props)
		}
		return err
	}

	app.Commands = []cli.Command{
		{
			Name:      "create",
			ArgsUsage: "[maxSize]",
			Action: func(c *cli.Context) {
				maxSize, err := strconv.ParseInt(c.Args().First(), 10, 64)
				if err != nil {
					maxSize = 2
				}
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.CreateTreeRequest{MaxSize: maxSize})
			},
		},
		{
			Name:      "insert",
			ArgsUsage: "key value",
			Action: func(c *cli.Context) {
				key, err := strconv.ParseInt(c.Args().First(), 10, 64)
				if err != nil {
					panic(err)
				}
				value := c.Args().Tail()[0]
				handleCredentialsFromCliContext(c)
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.InsertRequest{
					Credentials: &messages.Credentials{
						Token: c.GlobalString(globalFlagToken),
						Id:    c.GlobalInt64(globalFlagID),
					},
					Item: &messages.Item{
						Key:   key,
						Value: value,
					},
				})
			},
		},
		{
			Name:      "search",
			ArgsUsage: "key",
			Action: func(c *cli.Context) {
				key, err := strconv.ParseInt(c.Args().First(), 10, 64)
				if err != nil {
					panic(err)
				}
				handleCredentialsFromCliContext(c)
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.SearchRequest{
					Credentials: &messages.Credentials{
						Token: c.GlobalString(globalFlagToken),
						Id:    c.GlobalInt64(globalFlagID),
					},
					Key: key,
				})
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "key",
			Action: func(c *cli.Context) {
				key, err := strconv.ParseInt(c.Args().First(), 10, 64)
				if err != nil {
					panic(err)
				}
				handleCredentialsFromCliContext(c)
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.DeleteRequest{
					Credentials: &messages.Credentials{
						Token: c.GlobalString(globalFlagToken),
						Id:    c.GlobalInt64(globalFlagID),
					},
					Key: key,
				})
			},
		},
		{
			Name: "traverse",
			Action: func(c *cli.Context) {
				handleCredentialsFromCliContext(c)
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.TraverseRequest{
					Credentials: &messages.Credentials{
						Token: c.GlobalString(globalFlagToken),
						Id:    c.GlobalInt64(globalFlagID),
					},
				})
			},
		},
	}
	_ = app.Run(os.Args)
}
