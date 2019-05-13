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
		log.Panic("Missing credentials.")
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
	app.Usage = "proto.actor client for treeservice"
	app.UsageText = "treecli [global options] command [arguments...]"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "bind",
			Usage:       "address treecli should use",
			Value:       "localhost:8091",
			Destination: &bindAddr,
		},
		cli.StringFlag{
			Name:        "remote",
			Usage:       "address of the treeservice",
			Value:       "localhost:8090",
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

	before := func(c *cli.Context) error {
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
			HelpName: "create",
			Name:     "create",
			Usage:    "create a new search tree",
			Description: "Create a new search tree with the specified maximum size for its leafs (default 2). " +
				"Outputs id and token of the created tree.",
			ArgsUsage: "[maxSize=2]",
			Before:    before,
			Action: func(c *cli.Context) {
				maxSize, err := strconv.ParseInt(c.Args().First(), 10, 64)
				if err != nil {
					maxSize = 2
				}
				requestAndWait(rootContext, &wg, remotePid, pid, &messages.CreateTreeRequest{MaxSize: maxSize})
			},
		},
		{
			HelpName: "insert",
			Name:     "insert",
			Usage:    "insert key-value pair into tree",
			Description: "Inserts new key-value pair into specified tree. Outputs key-value pair on success. \n" +
				"   Fails if the specified tree doesn't exist or if an invalid token is provided.\n" +
				"   Also fails if the specified key already exists. " +
				"In this case the existing key-value pair will be printed.",
			ArgsUsage: "key value",
			Before:    before,
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
			HelpName:  "search",
			Name:      "search",
			ArgsUsage: "key",
			Usage:     "search value specified by key in tree",
			Description: "Searches value specified by key in specified tree. Outputs key-value pair if found. \n" +
				"   Fails if the specified tree doesn't exist or if an invalid token is provided.\n" +
				"   Also fails if the specified key doesn't exist. ",
			Before: before,
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
			HelpName:  "deleteitem",
			Name:      "deleteitem",
			ArgsUsage: "key",
			Usage:     "delete key-value pair in tree",
			Description: "Deletes key-value pair specified by key in specified tree. " +
				"Outputs deleted key-value pair on success. \n" +
				"   Fails if the specified tree doesn't exist or if an invalid token is provided.\n" +
				"   Also fails if the specified key doesn't exist. ",
			Before: before,
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
			HelpName: "traverse",
			Name:     "traverse",
			Usage:    "get all key-value pairs sorted by key",
			Description: "Gets all key-value pairs in specified tree sorted by keys. \n" +
				"   Fails if the specified tree doesn't exist or if an invalid token is provided.",
			Before: before,
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
			HelpName: "deletetree",
			Name:     "deletetree",
			Usage:    "remove tree from treeservice",
			Description: "Removes specified tree. Asks for confirmation by repeating the token.\n" +
				"   Fails if the specified tree doesn't exist or if an invalid token is provided.",
			Before: before,
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
