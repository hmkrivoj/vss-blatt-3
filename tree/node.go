package tree

import (
	"log"
	"sort"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
)

type nodeActor struct {
	left, right             *actor.PID
	content                 map[int]string
	maxSize, maxLeftSideKey int
	behaviour               actor.Behavior
}

func (state *nodeActor) Receive(context actor.Context) {
	state.behaviour.Receive(context)
}

func (state *nodeActor) leaf(context actor.Context) {
	switch msg := context.Message().(type) {
	case *messages.CreateTreeRequest:
		state.maxSize = int(msg.MaxSize)
		state.content = make(map[int]string)
		log.Printf("%s created", context.Self().Id)
	case *messages.InsertRequest:
		log.Printf("%s receives (%d, %s)", context.Self().Id, msg.Key, msg.Value)
		if _, exists := state.content[int(msg.Key)]; exists {
			context.Respond(messages.InsertResponse{Key: msg.Key, Type: messages.KEY_ALREADY_EXISTS})
		} else {
			state.content[int(msg.Key)] = msg.Value
			context.Respond(messages.InsertResponse{Key: msg.Key, Type: messages.SUCCESS})
		}
		if len(state.content) > state.maxSize {
			state.split(context)
		}
	case *messages.SearchRequest:
		if value, exists := state.content[int(msg.Key)]; exists {
			context.Respond(messages.SearchResponse{Key: msg.Key, Value: value, Type: messages.SUCCESS})
		} else {
			context.Respond(messages.SearchResponse{Key: msg.Key, Type: messages.NO_SUCH_KEY})
		}
	}
}

func (state *nodeActor) internalNode(context actor.Context) {
	switch msg := context.Message().(type) {
	case actor.PoisonPill:
		context.Poison(state.left)
		context.Poison(state.right)
	case messages.InsertRequest:
		if int(msg.Key) > state.maxLeftSideKey {
			log.Printf("%s forwards (%d, %s) to righthand child", context.Self().Id, msg.Key, msg.Value)
			context.Forward(state.right)
		} else {
			log.Printf("%s forwards (%d, %s) to lefthand child", context.Self().Id, msg.Key, msg.Value)
			context.Forward(state.left)
		}
	case messages.SearchRequest:
		if int(msg.Key) > state.maxLeftSideKey {
			context.Forward(state.right)
		} else {
			context.Forward(state.left)
		}
	}
}

func NodeActorProducer() actor.Actor {
	node := &nodeActor{behaviour: actor.NewBehavior()}
	node.behaviour.Become(node.leaf)
	return node
}

func (state *nodeActor) split(context actor.Context) {
	keys := state.sortedKeys()
	mid := len(keys) / 2
	state.maxLeftSideKey = keys[mid-1]
	log.Printf("%s splitting up - maximum key of lefthand child will be %d", context.Self().Id, state.maxLeftSideKey)

	log.Printf("%s creating lefthand child", context.Self().Id)
	state.left = context.Spawn(actor.PropsFromProducer(NodeActorProducer))
	context.Send(state.left, messages.CreateTreeRequest{MaxSize: int64(state.maxSize)})
	log.Printf("%s sending items to lefthand child: %v", context.Self().Id, keys[:mid])
	for _, key := range keys[:mid] {
		context.Send(state.left, messages.InsertRequest{Key: int64(key), Value: state.content[key]})
	}

	log.Printf("%s creating righthand child", context.Self().Id)
	state.right = context.Spawn(actor.PropsFromProducer(NodeActorProducer))
	context.Send(state.right, messages.CreateTreeRequest{MaxSize: int64(state.maxSize)})
	log.Printf("%s sending items to righthand child: %v", context.Self().Id, keys[mid:])
	for _, key := range keys[mid:] {
		context.Send(state.right, messages.InsertRequest{Key: int64(key), Value: state.content[key]})
	}

	state.content = make(map[int]string)
	state.behaviour.Become(state.internalNode)
}

func (state *nodeActor) sortedKeys() []int {
	keys := make([]int, 0)
	for key := range state.content {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
