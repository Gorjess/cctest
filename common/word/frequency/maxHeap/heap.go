package maxHeap

import "fmt"

type IEntry interface {
	Value() interface{}
	String() string
}

type Node struct {
	value IEntry
	left  *Node
	right *Node
}

func (n *Node) String() string {
	if n == nil || n.value == nil {
		return "n/a"
	}
	return n.value.String()
}

// print heap in pre-order,
func Literal(node *Node) string {
	if node == nil {
		return ""
	}
	s := fmt.Sprintf("[%s]-", node.String())

	s += Literal(node.left)
	s += Literal(node.right)

	return s
}

// form a complete binary tree from a given slice
func completeBT(input []IEntry, node *Node, startIdx, fullLen int) {
	if len(input) == 0 || node == nil || startIdx >= fullLen {
		return
	}

	node.value = input[startIdx]
	node.left = new(Node)
	node.right = new(Node)
	completeBT(input, node.left, 2*startIdx+1, fullLen)
	completeBT(input, node.right, 2*startIdx+2, fullLen)
}

// fromSlice builds a max-heap from a given slice
// and return the root node of the heap.
func FromSlice(input []IEntry) *Node {
	// 1. build a complete binary tree
	var (
		node = new(Node)
	)
	completeBT(input, node, 0, len(input))
	return node
}
