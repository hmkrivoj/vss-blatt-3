package tree

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
	"log"
	"sort"
)

type NodeActor struct {
	left, right             *actor.PID
	content                 map[int]string
	maxSize, maxLeftSideKey int
	behaviour               actor.Behavior
	credentials             messages.Credentials
}

func (state *NodeActor) Receive(context actor.Context) {
	state.behaviour.Receive(context)
}

func (state *NodeActor) leaf(context actor.Context) {
	switch msg := context.Message().(type) {
	case messages.InitNode:
		log.Printf("%s created", context.Self().Id)
		state.maxSize = int(msg.MaxSize)
		state.content = make(map[int]string)
		state.credentials = *msg.Credentials
	case messages.InsertRequest:
		log.Printf("%s receives (%d, %s)", context.Self().Id, msg.Key, msg.Value)
		if _, exists := state.content[int(msg.Key)]; state.credentials == *msg.Credentials && exists {
			context.Respond(messages.InsertResponse{Key: msg.Key, Success: false})
		} else {
			state.content[int(msg.Key)] = msg.Value
			context.Respond(messages.InsertResponse{Key: msg.Key, Success: true})
		}
		if len(state.content) > state.maxSize {
			state.split(context)
		}
	case messages.SearchRequest:
		if value, ok := state.content[int(msg.Key)]; state.credentials == *msg.Credentials && ok {
			context.Respond(messages.SearchResponse{Success: true, Key: msg.Key, Value: value})
		} else {
			context.Respond(messages.SearchResponse{Success: false, Key: msg.Key})
		}
	}
}

func (state *NodeActor) internalNode(context actor.Context) {
	switch msg := context.Message().(type) {
	case messages.InsertRequest:
		if state.credentials != *msg.Credentials {
			context.Respond(messages.InsertResponse{Key: msg.Key, Success: false})
			return
		}
		if int(msg.Key) > state.maxLeftSideKey {
			log.Printf("%s forwards (%d, %s) to righthand child", context.Self().Id, msg.Key, msg.Value)
			context.Forward(state.right)
		} else {
			log.Printf("%s forwards (%d, %s) to lefthand child", context.Self().Id, msg.Key, msg.Value)
			context.Forward(state.left)
		}
	case messages.SearchRequest:
		if state.credentials != *msg.Credentials {
			context.Respond(messages.SearchResponse{Key: msg.Key, Success: false})
			return
		}
		if int(msg.Key) > state.maxLeftSideKey {
			context.Forward(state.right)
		} else {
			context.Forward(state.left)
		}
	}
}

func nodeActorProducer() actor.Actor {
	node := &NodeActor{behaviour: actor.NewBehavior()}
	node.behaviour.Become(node.leaf)
	return node
}

func (state *NodeActor) split(context actor.Context) {
	keys := state.sortedKeys()
	mid := len(keys) / 2
	state.maxLeftSideKey = keys[mid-1]
	log.Printf("%s splitting up - maximum key of lefthand child will be %d", context.Self().Id, state.maxLeftSideKey)

	log.Printf("%s creating lefthand child", context.Self().Id)
	state.left = context.Spawn(actor.PropsFromProducer(nodeActorProducer))
	context.Send(state.left, messages.InitNode{MaxSize: int64(state.maxSize), Credentials: &state.credentials})
	log.Printf("%s sending items to lefthand child: %v", context.Self().Id, keys[:mid])
	for _, key := range keys[:mid] {
		context.Send(state.left, messages.InsertRequest{Key: int64(key), Value: state.content[key], Credentials: &state.credentials})
	}

	log.Printf("%s creating righthand child", context.Self().Id)
	state.right = context.Spawn(actor.PropsFromProducer(nodeActorProducer))
	context.Send(state.right, messages.InitNode{MaxSize: int64(state.maxSize), Credentials: &state.credentials})
	log.Printf("%s sending items to righthand child: %v", context.Self().Id, keys[mid:])
	for _, key := range keys[mid:] {
		context.Send(state.right, messages.InsertRequest{Key: int64(key), Value: state.content[key], Credentials: &state.credentials})
	}

	state.content = make(map[int]string)
	state.behaviour.Become(state.internalNode)
}

func (state *NodeActor) sortedKeys() []int {
	keys := make([]int, 0)
	for key := range state.content {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
