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
}

func (state *NodeActor) Receive(context actor.Context) {
	state.behaviour.Receive(context)
}

func (state *NodeActor) leaf(context actor.Context) {
	switch msg := context.Message().(type) {
	case messages.Create:
		log.Printf("%s created", context.Self().Id)
		state.maxSize = int(msg.MaxSize)
		state.content = make(map[int]string)
	case messages.Insert:
		log.Printf("%s receives (%d, %s)", context.Self().Id, msg.Key, msg.Value)
		state.content[int(msg.Key)] = msg.Value
		if len(state.content) > state.maxSize {
			state.split(context)
		}
	case messages.Search:
		value, ok := state.content[int(msg.Key)]
		context.Respond(messages.Found{HasFound: ok, Key: msg.Key, Value: value})
	}
}

func (state *NodeActor) internalNode(context actor.Context) {
	switch msg := context.Message().(type) {
	case messages.Insert:
		if int(msg.Key) > state.maxLeftSideKey {
			log.Printf("%s forwards (%d, %s) to righthand child", context.Self().Id, msg.Key, msg.Value)
			context.Forward(state.right)
		} else {
			log.Printf("%s forwards (%d, %s) to lefthand child", context.Self().Id, msg.Key, msg.Value)
			context.Forward(state.left)
		}
	case messages.Search:
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
	context.Send(state.left, messages.Create{MaxSize: int32(state.maxSize)})
	log.Printf("%s sending items to lefthand child: %v", context.Self().Id, keys[:mid])
	for _, key := range keys[:mid] {
		context.Send(state.left, messages.Insert{Key: int32(key), Value: state.content[key]})
	}

	log.Printf("%s creating righthand child", context.Self().Id)
	state.right = context.Spawn(actor.PropsFromProducer(nodeActorProducer))
	context.Send(state.right, messages.Create{MaxSize: int32(state.maxSize)})
	log.Printf("%s sending items to righthand child: %v", context.Self().Id, keys[mid:])
	for _, key := range keys[mid:] {
		context.Send(state.right, messages.Insert{Key: int32(key), Value: state.content[key]})
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
