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
const globalFlagID = "id"
const globalFlagToken = "token"

const commandCreatetreeName = "createtree"
const commandCreatetreeFlagMaxsize = "maxsize"

const commandInsertName = "insert"
const commandInsertFlagKey = "key"
const commandInsertFlagValue = "value"

const commandSearchName = "search"
const commandSearchFlagKey = "key"

const commandDeleteName = "delete"
const commandDeleteFlagKey = "key"

const commandTraverseName = "traverse"

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
	if !c.GlobalIsSet(globalFlagID) || !c.GlobalIsSet(globalFlagToken) {
		panic("Missing credentials.")
	}
}

func requestResult(pid *actor.PID, message interface{}) interface{} {
	result, err := actor.EmptyRootContext.RequestFuture(pid, message, timeout).Result()
	if err != nil {
		panic(err)
	}
	return result
}

func commandCreatetreeAction(c *cli.Context) error {
	pid := spawnRemoteFromCliContext(c)
	res := requestResult(pid, &messages.CreateTreeRequest{MaxSize: c.Int64(commandCreatetreeFlagMaxsize)})
	switch response := res.(type) {
	case *messages.CreateTreeResponse:
		fmt.Printf("id: %d, token: %s\n", response.Credentials.Id, response.Credentials.Token)
	default:
		panic("Wrong message type")
	}
	return nil
}

func commandInsertAction(c *cli.Context) error {
	if !c.IsSet(commandInsertFlagKey) || !c.IsSet(commandInsertFlagValue) {
		panic("Missing key or value.")
	}
	handleCredentialsFromCliContext(c)
	pid := spawnRemoteFromCliContext(c)
	res := requestResult(
		pid,
		&messages.InsertRequest{
			Credentials: &messages.Credentials{
				Token: c.GlobalString(globalFlagToken),
				Id:    c.GlobalInt64(globalFlagID),
			},
			Item: &messages.Item{
				Key:   c.Int64(commandInsertFlagKey),
				Value: c.String(commandInsertFlagValue),
			},
		},
	)
	switch msg := res.(type) {
	case *messages.InsertResponse:
		switch msg.Type {
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
	default:
		panic("Wrong message type")
	}
	return nil
}

func commandSearchAction(c *cli.Context) error {
	if !c.IsSet(commandSearchFlagKey) {
		panic("Missing key.")
	}
	handleCredentialsFromCliContext(c)
	pid := spawnRemoteFromCliContext(c)
	res := requestResult(
		pid,
		&messages.SearchRequest{
			Credentials: &messages.Credentials{
				Token: c.GlobalString(globalFlagToken),
				Id:    c.GlobalInt64(globalFlagID),
			},
			Key: c.Int64(commandSearchFlagKey),
		},
	)
	switch msg := res.(type) {
	case *messages.SearchResponse:
		switch msg.Type {
		case messages.SUCCESS:
			fmt.Printf("Value for key %d: %s\n", msg.Item.Key, msg.Item.Value)
		case messages.NO_SUCH_KEY:
			panic(fmt.Sprintf("Tree contains no key %d", c.Int64(commandSearchFlagKey)))
		case messages.ACCESS_DENIED:
			panic("Invalid credentials")
		case messages.NO_SUCH_TREE:
			panic("No such tree")
		default:
			panic("Unknown response type")
		}
	default:
		panic("Wrong message type")
	}
	return nil
}

func commandDeleteAction(c *cli.Context) error {
	if !c.IsSet(commandDeleteFlagKey) {
		panic("Missing key.")
	}
	handleCredentialsFromCliContext(c)
	pid := spawnRemoteFromCliContext(c)
	res := requestResult(
		pid,
		&messages.DeleteRequest{
			Credentials: &messages.Credentials{
				Token: c.GlobalString(globalFlagToken),
				Id:    c.GlobalInt64(globalFlagID),
			},
			Key: c.Int64(commandDeleteFlagKey),
		},
	)
	switch msg := res.(type) {
	case *messages.DeleteResponse:
		switch msg.Type {
		case messages.SUCCESS:
			fmt.Printf("Successfully deleted key %d from tree\n", c.Int64(commandDeleteFlagKey))
		case messages.NO_SUCH_KEY:
			panic(fmt.Sprintf("Tree contains no key %d", c.Int64(commandDeleteFlagKey)))
		case messages.ACCESS_DENIED:
			panic("Invalid credentials")
		case messages.NO_SUCH_TREE:
			panic("No such tree")
		default:
			panic("Unknown response type")
		}
	default:
		panic("Wrong message type")
	}
	return nil
}

func commandTraverseAction(c *cli.Context) error {
	handleCredentialsFromCliContext(c)
	pid := spawnRemoteFromCliContext(c)
	res := requestResult(
		pid,
		&messages.TraverseRequest{
			Credentials: &messages.Credentials{
				Token: c.GlobalString(globalFlagToken),
				Id:    c.GlobalInt64(globalFlagID),
			},
		},
	)
	switch msg := res.(type) {
	case *messages.TraverseResponse:
		switch msg.Type {
		case messages.SUCCESS:
			fmt.Printf("%q\n", msg.Items)
		case messages.ACCESS_DENIED:
			panic("Invalid credentials")
		case messages.NO_SUCH_TREE:
			panic("No such tree")
		default:
			panic("Unknown response type")
		}
	default:
		panic("Wrong message type")
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
			Name:  globalFlagID,
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
		{
			Name: commandDeleteName,
			Flags: []cli.Flag{
				cli.Int64Flag{
					Name: commandDeleteFlagKey,
				},
			},
			Action: commandDeleteAction,
		},
		{
			Name:   commandTraverseName,
			Action: commandTraverseAction,
		},
	}
	_ = app.Run(os.Args)
}
