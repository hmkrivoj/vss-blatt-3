package tree

import (
	"log"
	"sort"
	"time"

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
		log.Printf("%s receives (%d, %s)", context.Self().Id, msg.Item.Key, msg.Item.Value)
		if _, exists := state.content[int(msg.Item.Key)]; exists {
			context.Respond(&messages.InsertResponse{Type: messages.KEY_ALREADY_EXISTS})
		} else {
			state.content[int(msg.Item.Key)] = msg.Item.Value
			context.Respond(&messages.InsertResponse{Type: messages.SUCCESS})
		}
		if len(state.content) > state.maxSize {
			itemsLeft, maxLeftSideKey, itemsRight := split(state.content)
			state.left = createLeaf(context, int64(state.maxSize), itemsLeft)
			state.right = createLeaf(context, int64(state.maxSize), itemsRight)
			state.maxLeftSideKey = maxLeftSideKey
			for key := range state.content {
				delete(state.content, key)
			}
			state.behaviour.Become(state.internalNode)
		}
	case *messages.MultiInsert:
		for _, item := range msg.Items {
			state.content[int(item.Key)] = item.Value
		}
	case *messages.SearchRequest:
		if value, exists := state.content[int(msg.Key)]; exists {
			context.Respond(&messages.SearchResponse{Item: &messages.Item{Key: msg.Key, Value: value}, Type: messages.SUCCESS})
		} else {
			context.Respond(&messages.SearchResponse{Type: messages.NO_SUCH_KEY})
		}
	case *messages.DeleteRequest:
		if _, exists := state.content[int(msg.Key)]; exists {
			delete(state.content, int(msg.Key))
			context.Respond(&messages.DeleteResponse{Type: messages.SUCCESS})
		} else {
			context.Respond(&messages.DeleteResponse{Type: messages.NO_SUCH_KEY})
		}
	case *messages.TraverseRequest:
		context.Respond(&messages.TraverseResponse{Items: itemsSortedByKeys(state.content)})
	}
}

func (state *nodeActor) internalNode(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.PoisonPill:
		context.Poison(state.left)
		context.Poison(state.right)
	case *messages.InsertRequest:
		if int(msg.Item.Key) > state.maxLeftSideKey {
			log.Printf("%s forwards (%d, %s) to righthand child", context.Self().Id, msg.Item.Key, msg.Item.Value)
			context.Forward(state.right)
		} else {
			log.Printf("%s forwards (%d, %s) to lefthand child", context.Self().Id, msg.Item.Key, msg.Item.Value)
			context.Forward(state.left)
		}
	case *messages.SearchRequest:
		if int(msg.Key) > state.maxLeftSideKey {
			context.Forward(state.right)
		} else {
			context.Forward(state.left)
		}
	case *messages.DeleteRequest:
		if int(msg.Key) > state.maxLeftSideKey {
			context.Forward(state.right)
		} else {
			context.Forward(state.left)
		}
	case *messages.TraverseRequest:
		leftFuture := context.RequestFuture(state.left, &messages.TraverseRequest{}, 5*time.Second)
		rightFuture := context.RequestFuture(state.right, &messages.TraverseRequest{}, 5*time.Second)
		context.AwaitFuture(leftFuture, func(resLeft interface{}, errLeft error) {
			if errLeft != nil {
				panic(errLeft)
			}
			switch msgLeft := resLeft.(type) {
			case *messages.TraverseResponse:
				context.AwaitFuture(rightFuture, func(resRight interface{}, errRight error) {
					if errRight != nil {
						panic(errRight)
					}
					switch msgRight := resRight.(type) {
					case *messages.TraverseResponse:
						items := append(msgLeft.Items, msgRight.Items...)
						sort.Slice(items, func(i, j int) bool {
							return items[i].Key < items[j].Key
						})
						context.Respond(&messages.TraverseResponse{Items: items})
					default:
						panic("Wrong message from right child")
					}
				})
			default:
				panic("Wrong message from left child")
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
	context.Send(pid, &messages.CreateTreeRequest{MaxSize: int64(maxsize)})
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
