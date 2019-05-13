package main

import (
	"fmt"
	"log"
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

func assertCredentialsExist(c *cli.Context) {
	if !c.GlobalIsSet(globalFlagID) || !c.GlobalIsSet(globalFlagToken) {
		panic("Missing credentials.")
	}
}

func requestAndWait(
	context *actor.RootContext,
	wg *sync.WaitGroup,
	remotePid *actor.PID,
	pid *actor.PID,
	message interface{},
) {
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
		log.Printf("No tree with id %d", msg.Id)
	case *messages.NoSuchKeyError:
		c.Stop(c.Self())
		log.Printf("Tree contains no key %d", msg.Key)
	case *messages.InvalidTokenError:
		c.Stop(c.Self())
		log.Printf("Invalid token %s for tree %d", msg.Credentials.Token, msg.Credentials.Id)
	case *messages.KeyAlreadyExistsError:
		c.Stop(c.Self())
		log.Printf("Tree already contains item (%d, %s)", msg.Item.Key, msg.Item.Value)
	case *messages.CreateTreeResponse:
		c.Stop(c.Self())
		log.Printf("id: %d, token: %s", msg.Credentials.Id, msg.Credentials.Token)
	case *messages.InsertResponse:
		c.Stop(c.Self())
		log.Printf("(%d, %s) successfully inserted", msg.Item.Key, msg.Item.Value)
	case *messages.SearchResponse:
		c.Stop(c.Self())
		log.Printf("Found item (%d, %s)", msg.Item.Key, msg.Item.Value)
	case *messages.DeleteResponse:
		c.Stop(c.Self())
		log.Printf("Successfully deleted item (%d, %s) from tree", msg.Item.Key, msg.Item.Value)
	case *messages.TraverseResponse:
		for _, item := range msg.Items {
			log.Printf("(%d, %s)", item.Key, item.Value)
		}
		c.Stop(c.Self())
	case *actor.Stopped:
		state.wg.Done()
	case *messages.DeleteTreeResponse:
		c.Stop(c.Self())
		log.Printf("Successfully deleted tree %d", msg.Credentials.Id)
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
				assertCredentialsExist(c)
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
				assertCredentialsExist(c)
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
			Name:      "deleteitem",
			ArgsUsage: "key",
			Action: func(c *cli.Context) {
				key, err := strconv.ParseInt(c.Args().First(), 10, 64)
				if err != nil {
					panic(err)
				}
				assertCredentialsExist(c)
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
				assertCredentialsExist(c)
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.TraverseRequest{
					Credentials: &messages.Credentials{
						Token: c.GlobalString(globalFlagToken),
						Id:    c.GlobalInt64(globalFlagID),
					},
				})
			},
		},
		{
			Name: "deletetree",
			Action: func(c *cli.Context) {
				assertCredentialsExist(c)
				fmt.Printf("Repeat token to delete tree %d: ", c.GlobalInt64(globalFlagID))
				var token string
				n, err := fmt.Scanf("%s", &token)
				if n != 1 || err != nil || c.GlobalString(globalFlagToken) != token {
					fmt.Printf("Token doesn't match flag - tree %d remains", c.GlobalInt64(globalFlagID))
					return
				}
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.DeleteTreeRequest{
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
