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
	isLeaf                  bool
}

func (state *NodeActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case messages.Create:
		log.Printf("%s created", context.Self().Id)
		state.maxSize = int(msg.MaxSize)
		state.content = make(map[int]string)
		state.isLeaf = true
	case messages.Insert:
		if state.isLeaf {
			log.Printf("%s receives (%d, %s)", context.Self().Id, msg.Key, msg.Value)
			state.content[int(msg.Key)] = msg.Value
		} else {
			if int(msg.Key) > state.maxLeftSideKey {
				log.Printf("%s forwards (%d, %s) to righthand child", context.Self().Id, msg.Key, msg.Value)
				context.Forward(state.right)
			} else {
				log.Printf("%s forwards (%d, %s) to lefthand child", context.Self().Id, msg.Key, msg.Value)
				context.Forward(state.left)
			}
		}
	case messages.Search:
		if state.isLeaf {
			value, ok := state.content[int(msg.Key)]
			context.Respond(messages.Found{HasFound: ok, Key: msg.Key, Value: value})
		} else {
			if int(msg.Key) > state.maxLeftSideKey {
				context.Forward(state.right)
			} else {
				context.Forward(state.left)
			}
		}
	}
	if state.isLeaf && len(state.content) > state.maxSize {
		state.split(context)
	}
}

func nodeActorProducer() actor.Actor {
	return &NodeActor{}
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
	state.isLeaf = false
}

func (state *NodeActor) sortedKeys() []int {
	keys := make([]int, 0)
	for key := range state.content {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
