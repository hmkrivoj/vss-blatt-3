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

const timeout = 5 * time.Second

const globalFlagBind = "bind"
const globalFlagRemote = "remote"
const globalFlagId = "id"
const globalFlagToken = "token"

const commandCreatetreeName = "createtree"
const commandCreatetreeFlagMaxsize = "maxsize"

const commandInsertName = "insert"
const commandInsertFlagKey = "key"
const commandInsertFlagValue = "value"

const commandSearchName = "search"
const commandSearchFlagKey = "key"

func spawnRemoteFromCliContext(c *cli.Context) *actor.PID {
	remote.Start(c.GlobalString(globalFlagBind))
	pidResp, err := remote.SpawnNamed(
		c.GlobalString(globalFlagRemote),
		"remote",
		"treeservice",
		timeout,
	)
	if err != nil {
		panic(err)
	}
	pid := pidResp.Pid
	return pid
}

func handleCredentialsFromCliContext(c *cli.Context) {
	if !c.GlobalIsSet(globalFlagId) || !c.GlobalIsSet(globalFlagToken) {
		panic("Missing credentials.")
	}
}

func commandCreatetreeAction(c *cli.Context) error {
	pid := spawnRemoteFromCliContext(c)
	res, err := actor.EmptyRootContext.RequestFuture(
		pid,
		&messages.CreateTreeRequest{MaxSize: c.Int64(commandCreatetreeFlagMaxsize)},
		timeout,
	).Result()
	if err != nil {
		panic(err)
	}
	response := res.(*messages.CreateTreeResponse)
	fmt.Printf("id: %d, token: %s\n", response.Credentials.Id, response.Credentials.Token)
	return nil
}

func commandInsertAction(c *cli.Context) error {
	if !c.IsSet(commandInsertFlagKey) || !c.IsSet(commandInsertFlagValue) {
		panic("Missing key or value.")
	}
	handleCredentialsFromCliContext(c)
	pid := spawnRemoteFromCliContext(c)
	res, err := actor.EmptyRootContext.RequestFuture(
		pid,
		&messages.InsertRequest{
			Credentials: &messages.Credentials{
				Token: c.GlobalString(globalFlagToken),
				Id:    c.GlobalInt64(globalFlagId),
			},
			Key:   c.Int64(commandInsertFlagKey),
			Value: c.String(commandInsertFlagValue),
		},
		timeout,
	).Result()
	if err != nil {
		panic(err)
	}
	response := res.(*messages.InsertResponse)
	switch response.Type {
	case messages.SUCCESS:
		fmt.Printf("(%d, %s) successfully inserted\n", c.Int64(commandInsertFlagKey), c.String(commandInsertFlagValue))
	case messages.KEY_ALREADY_EXISTS:
		panic(fmt.Sprintf("Tree already contains key %d", c.Int64(commandInsertFlagKey)))
	case messages.ACCESS_DENIED:
		panic("Invalid credentials")
	case messages.NO_SUCH_TREE:
		panic("No such tree")
	default:
		panic("Unknown response type")
	}
	return nil
}

func commandSearchAction(c *cli.Context) error {
	if !c.IsSet(commandSearchFlagKey) {
		panic("Missing key.")
	}
	handleCredentialsFromCliContext(c)
	pid := spawnRemoteFromCliContext(c)
	res, err := actor.EmptyRootContext.RequestFuture(
		pid,
		&messages.SearchRequest{
			Credentials: &messages.Credentials{
				Token: c.GlobalString(globalFlagToken),
				Id:    c.GlobalInt64(globalFlagId),
			},
			Key: c.Int64(commandSearchFlagKey),
		},
		timeout,
	).Result()
	if err != nil {
		panic(err)
	}
	response := res.(*messages.SearchResponse)
	switch response.Type {
	case messages.SUCCESS:
		fmt.Printf("Value for key %d: %s\n", c.Int64(commandSearchFlagKey), response.Value)
	case messages.NO_SUCH_KEY:
		panic(fmt.Sprintf("Tree contains no key %d", c.Int64(commandSearchFlagKey)))
	case messages.ACCESS_DENIED:
		panic("Invalid credentials")
	case messages.NO_SUCH_TREE:
		panic("No such tree")
	default:
		panic("Unknown response type")
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  globalFlagBind,
			Usage: "address treecli should use",
			Value: "treecli.actors:8091",
		},
		cli.StringFlag{
			Name:  globalFlagRemote,
			Usage: "address of the treeservice",
			Value: "treeservice.actors:8090",
		},
		cli.Int64Flag{
			Name:  globalFlagId,
			Usage: "id of the tree you want to alter",
		},
		cli.StringFlag{
			Name:  globalFlagToken,
			Usage: "token to authorize your access for the specified tree",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: commandCreatetreeName,
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name:  commandCreatetreeFlagMaxsize,
					Usage: "max size of a leaf",
					Value: 2,
				},
			},
			Action: commandCreatetreeAction,
		},
		{
			Name: commandInsertName,
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name: commandInsertFlagKey,
				},
				cli.StringFlag{
					Name: commandInsertFlagValue,
				},
			},
			Action: commandInsertAction,
		},
		{
			Name: commandSearchName,
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name: commandSearchFlagKey,
				},
			},
			Action: commandSearchAction,
		},
	}
	_ = app.Run(os.Args)
}
