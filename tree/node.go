package tree

import (
	"github.com/AsynkronIT/protoactor-go/actor"
)

type NodeActor struct {
	left, right NodeActor
	content     map[int]string
}

func (state *NodeActor) Receive(context actor.Context) {

}
