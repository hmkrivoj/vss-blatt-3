package tree

import (
	"log"
	"sort"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ob-vss-ss19/blatt-3-forever_alone/messages"
)

// Actor for nodes. Implements actor.Actor.
type nodeActor struct {
	left, right             *actor.PID
	content                 map[int]string
	maxSize, maxLeftSideKey int
	behaviour               actor.Behavior
}

// Receives messages.
func (state *nodeActor) Receive(context actor.Context) {
	state.behaviour.Receive(context)
}

// Behaviour for leafs
func (state *nodeActor) leaf(context actor.Context) {
	name := context.Self().Id
	switch msg := context.Message().(type) {
	case *messages.CreateTreeRequest:
		// Init leaf
		state.maxSize = int(msg.MaxSize)
		state.content = make(map[int]string)
		log.Printf("Leaf %s created with maxSize %d", name, state.maxSize)
	case *messages.InsertRequest:
		log.Printf("Leaf %s receives (%d, %s)", name, msg.Item.Key, msg.Item.Value)
		if value, exists := state.content[int(msg.Item.Key)]; exists {
			log.Printf("Leaf %s already contains pair with same key: (%d, %s)", name, msg.Item.Key, value)
			context.Respond(&messages.KeyAlreadyExistsError{Item: &messages.Item{Key: msg.Item.Key, Value: value}})
		} else {
			state.content[int(msg.Item.Key)] = msg.Item.Value
			log.Printf("Leaf %s saved (%d, %s)", name, msg.Item.Key, msg.Item.Value)
			context.Respond(&messages.InsertResponse{Item: msg.Item})
		}
		if len(state.content) > state.maxSize {
			log.Printf("Leaf %s too big - splitting up", name)
			itemsLeft, maxLeftSideKey, itemsRight := split(state.content)
			state.left = createLeaf(context, int64(state.maxSize), itemsLeft)
			state.right = createLeaf(context, int64(state.maxSize), itemsRight)
			state.maxLeftSideKey = maxLeftSideKey
			for key := range state.content {
				delete(state.content, key)
			}
			state.behaviour.Become(state.internalNode)
			log.Printf("Leaf %s became internalNode with maxLeftSideKey = %d", name, state.maxLeftSideKey)
		}
	case *messages.MultiInsert:
		// This message type is used when a new leaf must be filled after splitting an internal node up
		for _, item := range msg.Items {
			state.content[int(item.Key)] = item.Value
			log.Printf("Leaf %s saved (%d, %s)", name, item.Key, item.Value)
		}
	case *messages.SearchRequest:
		if value, exists := state.content[int(msg.Key)]; exists {
			log.Printf("Leaf %s contains searched key %d: (%d, %s)", name, msg.Key, msg.Key, value)
			context.Respond(&messages.SearchResponse{Item: &messages.Item{Key: msg.Key, Value: value}})
		} else {
			log.Printf("Leaf %s does not contain searched key %d -> There is no key %d in this tree", name, msg.Key, msg.Key)
			context.Respond(&messages.NoSuchKeyError{Key: msg.Key})
		}
	case *messages.DeleteRequest:
		if value, exists := state.content[int(msg.Key)]; exists {
			log.Printf("Leaf %s contains key to be deleted. Deleting (%d, %s)", name, msg.Key, value)
			delete(state.content, int(msg.Key))
			context.Respond(&messages.DeleteResponse{Item: &messages.Item{Key: msg.Key, Value: value}})
		} else {
			log.Printf("Leaf %s does not contain key to be deleted: %d. Do nothing", name, msg.Key)
			context.Respond(&messages.NoSuchKeyError{Key: msg.Key})
		}
	case *messages.TraverseRequest:
		log.Printf("Leaf %s responding with its sorted items", name)
		context.Respond(&messages.TraverseResponse{Items: itemsSortedByKeys(state.content)})
	}
}

func (state *nodeActor) internalNode(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.PoisonPill:
		log.Printf("Internal node %s poisoned. Poisoning children.", context.Self().Id)
		context.Poison(state.left)
		context.Poison(state.right)
	case *messages.InsertRequest:
		if int(msg.Item.Key) > state.maxLeftSideKey {
			log.Printf("Internal node %s forwards (%d, %s) to righthand child", context.Self().Id, msg.Item.Key, msg.Item.Value)
			context.Forward(state.right)
		} else {
			log.Printf("Internal node %s forwards (%d, %s) to lefthand child", context.Self().Id, msg.Item.Key, msg.Item.Value)
			context.Forward(state.left)
		}
	case *messages.SearchRequest:
		if int(msg.Key) > state.maxLeftSideKey {
			log.Printf("Internal node %s forwards search request for key %d to righthand child (bigger than %d)", context.Self().Id, msg.Key, state.maxLeftSideKey)
			context.Forward(state.right)
		} else {
			log.Printf("Internal node %s forwards search request for key %d to lefthand child (equal or smaller than %d)", context.Self().Id, msg.Key, state.maxLeftSideKey)
			context.Forward(state.left)
		}
	case *messages.DeleteRequest:
		if int(msg.Key) > state.maxLeftSideKey {
			log.Printf("Internal node %s forwards delete request for key %d to righthand child (bigger than %d)", context.Self().Id, msg.Key, state.maxLeftSideKey)
			context.Forward(state.right)
		} else {
			log.Printf("Internal node %s forwards delete request for key %d to lefthand child (equal or smaller than %d)", context.Self().Id, msg.Key, state.maxLeftSideKey)
			context.Forward(state.left)
		}
	case *messages.TraverseRequest:
		leftFuture := context.RequestFuture(state.left, &messages.TraverseRequest{}, 5*time.Second)
		rightFuture := context.RequestFuture(state.right, &messages.TraverseRequest{}, 5*time.Second)
		log.Printf("Internal node %s fires traverserequests to its children", context.Self().Id)
		context.AwaitFuture(leftFuture, func(resLeft interface{}, errLeft error) {
			if errLeft != nil {
				log.Panic(errLeft)
			}
			switch msgLeft := resLeft.(type) {
			case *messages.TraverseResponse:
				log.Printf("Left future fired by internal node %s arrived", context.Self().Id)
				context.AwaitFuture(rightFuture, func(resRight interface{}, errRight error) {
					if errRight != nil {
						log.Panic(errRight)
					}
					log.Printf("Right future fired by internal node %s arrived", context.Self().Id)
					switch msgRight := resRight.(type) {
					case *messages.TraverseResponse:
						log.Printf("Merging results of futures fired by internal node %s", context.Self().Id)
						items := append(msgLeft.Items, msgRight.Items...)
						sort.Slice(items, func(i, j int) bool {
							return items[i].Key < items[j].Key
						})
						context.Respond(&messages.TraverseResponse{Items: items})
					default:
						log.Panicf("Right future fired by internal node %s arrived in unknown type", context.Self().Id)
					}
				})
			default:
				log.Panicf("Left future fired by internal node %s arrived in unknown type", context.Self().Id)
			}
		})
	}
}

func NodeActorProducer() actor.Actor {
	node := &nodeActor{behaviour: actor.NewBehavior()}
	node.behaviour.Become(node.leaf)
	return node
}

func createLeaf(context actor.Context, maxsize int64, items []*messages.Item) *actor.PID {
	pid := context.Spawn(actor.PropsFromProducer(NodeActorProducer))
	context.Send(pid, &messages.CreateTreeRequest{MaxSize: maxsize})
	context.Send(pid, &messages.MultiInsert{Items: items})
	return pid
}

func split(content map[int]string) ([]*messages.Item, int, []*messages.Item) {
	items := itemsSortedByKeys(content)
	mid := len(items) / 2
	return items[:mid], int(items[mid-1].Key), items[mid:]
}

func itemsSortedByKeys(content map[int]string) []*messages.Item {
	items := make([]*messages.Item, 0)
	for key, value := range content {
		items = append(items, &messages.Item{Key: int64(key), Value: value})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Key < items[j].Key
	})
	return items
}
