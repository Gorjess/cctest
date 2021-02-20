package maxHeap

import (
	"sort"
)

type Interface interface {
	sort.Interface
}

func heapify(input Interface, i, n int) {
	var (
		root  = i
		left  = 2*i + 1
		right = 2*i + 2
	)

	// find largest value
	if left < n && !input.Less(left, root) {
		root = left
	}
	if right < n && !input.Less(right, root) {
		root = right
	}

	// swap needed
	if root != i {
		input.Swap(root, i)

		// recursivly heapify the affected children
		heapify(input, root, n)
	}
}

func New(input Interface) {
	n := input.Len()

	// heapify from bottom to top
	for i := n/2 - 1; i >= 0; i-- {
		heapify(input, i, n)
	}
}
