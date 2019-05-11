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

const globalFlagID = "id, i"
const globalFlagToken = "token, t"

func handleCredentialsFromCliContext(c *cli.Context) {
	if !c.GlobalIsSet(globalFlagID) || !c.GlobalIsSet(globalFlagToken) {
		panic("Missing credentials.")
	}
}

type treeCliActor struct {
	wg sync.WaitGroup
}

func (*treeCliActor) Receive(c actor.Context) {
	switch msg := c.Message().(type) {
	case *messages.NoSuchTreeError:
		panic(fmt.Sprintf("No tree with id %d", msg.Id))
	case *messages.NoSuchKeyError:
		panic(fmt.Sprintf("Tree contains no key %d", msg.Key))
	case *messages.InvalidTokenError:
		panic(fmt.Sprintf("Invalid token %s for tree %d", msg.Credentials.Token, msg.Credentials.Id))
	case *messages.KeyAlreadyExistsError:
		panic(fmt.Sprintf("Tree already contains item (%d, %s)", msg.Item.Key, msg.Item.Value))
	case *messages.CreateTreeResponse:
		fmt.Printf("id: %d, token: %s\n", msg.Credentials.Id, msg.Credentials.Token)
	case *messages.InsertResponse:
		fmt.Printf("(%d, %s) successfully inserted\n", msg.Item.Key, msg.Item.Value)
	case *messages.SearchResponse:
		fmt.Printf("Found item (%d, %s)\n", msg.Item.Key, msg.Item.Value)
	case *messages.DeleteResponse:
		fmt.Printf("Successfully deleted item (%d, %s) from tree\n", msg.Item.Key, msg.Item.Value)
	case *messages.TraverseResponse:
		for _, item := range msg.Items {
			fmt.Printf("(%d, %s), ", item.Key, item.Value)
		}
		fmt.Println()
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
			Name:        "bind, b",
			Usage:       "address treecli should use",
			Value:       "treecli.actors:8091",
			Destination: &bindAddr,
		},
		cli.StringFlag{
			Name:        "remote, r",
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
			myActor := treeCliActor{wg: wg}
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
				wg.Add(1)
				rootContext.RequestWithCustomSender(
					remotePid,
					&messages.CreateTreeRequest{
						MaxSize: maxSize,
					},
					pid,
				)
				wg.Wait()
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
				wg.Add(1)
				rootContext.RequestWithCustomSender(
					remotePid,
					&messages.RequestWithCredentials{
						Credentials: &messages.Credentials{
							Token: c.GlobalString(globalFlagToken),
							Id:    c.GlobalInt64(globalFlagID),
						},
						Request: &messages.RequestWithCredentials_Insert{
							Insert: &messages.InsertRequest{
								Item: &messages.Item{
									Key:   key,
									Value: value,
								},
							},
						},
					},
					pid,
				)
				wg.Wait()
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
				wg.Add(1)
				rootContext.RequestWithCustomSender(
					remotePid,
					&messages.RequestWithCredentials{
						Credentials: &messages.Credentials{
							Token: c.GlobalString(globalFlagToken),
							Id:    c.GlobalInt64(globalFlagID),
						},
						Request: &messages.RequestWithCredentials_Search{
							Search: &messages.SearchRequest{
								Key: key,
							},
						},
					},
					pid,
				)
				wg.Wait()
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
				wg.Add(1)
				rootContext.RequestWithCustomSender(
					remotePid,
					&messages.RequestWithCredentials{
						Credentials: &messages.Credentials{
							Token: c.GlobalString(globalFlagToken),
							Id:    c.GlobalInt64(globalFlagID),
						},
						Request: &messages.RequestWithCredentials_Delete{
							Delete: &messages.DeleteRequest{
								Key: key,
							},
						},
					},
					pid,
				)
				wg.Wait()
			},
		},
		{
			Name: "traverse",
			Action: func(c *cli.Context) {
				handleCredentialsFromCliContext(c)
				wg.Add(1)
				rootContext.RequestWithCustomSender(
					remotePid,
					&messages.RequestWithCredentials{
						Credentials: &messages.Credentials{
							Token: c.GlobalString(globalFlagToken),
							Id:    c.GlobalInt64(globalFlagID),
						},
						Request: &messages.RequestWithCredentials_Traverse{Traverse: &messages.TraverseRequest{}},
					},
					pid,
				)
				wg.Wait()
			},
		},
	}
	_ = app.Run(os.Args)
}
