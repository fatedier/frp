/**
 * A thread-safe tree which caches inverted matrices.
 *
 * Copyright 2016, Peter Collins
 */

package reedsolomon

import (
	"errors"
	"sync"
)

// The tree uses a Reader-Writer mutex to make it thread-safe
// when accessing cached matrices and inserting new ones.
type inversionTree struct {
	mutex *sync.RWMutex
	root  inversionNode
}

type inversionNode struct {
	matrix   matrix
	children []*inversionNode
}

// newInversionTree initializes a tree for storing inverted matrices.
// Note that the root node is the identity matrix as it implies
// there were no errors with the original data.
func newInversionTree(dataShards, parityShards int) inversionTree {
	identity, _ := identityMatrix(dataShards)
	root := inversionNode{
		matrix:   identity,
		children: make([]*inversionNode, dataShards+parityShards),
	}
	return inversionTree{
		mutex: &sync.RWMutex{},
		root:  root,
	}
}

// GetInvertedMatrix returns the cached inverted matrix or nil if it
// is not found in the tree keyed on the indices of invalid rows.
func (t inversionTree) GetInvertedMatrix(invalidIndices []int) matrix {
	// Lock the tree for reading before accessing the tree.
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// If no invalid indices were give we should return the root
	// identity matrix.
	if len(invalidIndices) == 0 {
		return t.root.matrix
	}

	// Recursively search for the inverted matrix in the tree, passing in
	// 0 as the parent index as we start at the root of the tree.
	return t.root.getInvertedMatrix(invalidIndices, 0)
}

// errAlreadySet is returned if the root node matrix is overwritten
var errAlreadySet = errors.New("the root node identity matrix is already set")

// InsertInvertedMatrix inserts a new inverted matrix into the tree
// keyed by the indices of invalid rows.  The total number of shards
// is required for creating the proper length lists of child nodes for
// each node.
func (t inversionTree) InsertInvertedMatrix(invalidIndices []int, matrix matrix, shards int) error {
	// If no invalid indices were given then we are done because the
	// root node is already set with the identity matrix.
	if len(invalidIndices) == 0 {
		return errAlreadySet
	}

	if !matrix.IsSquare() {
		return errNotSquare
	}

	// Lock the tree for writing and reading before accessing the tree.
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Recursively create nodes for the inverted matrix in the tree until
	// we reach the node to insert the matrix to.  We start by passing in
	// 0 as the parent index as we start at the root of the tree.
	t.root.insertInvertedMatrix(invalidIndices, matrix, shards, 0)

	return nil
}

func (n inversionNode) getInvertedMatrix(invalidIndices []int, parent int) matrix {
	// Get the child node to search next from the list of children.  The
	// list of children starts relative to the parent index passed in
	// because the indices of invalid rows is sorted (by default).  As we
	// search recursively, the first invalid index gets popped off the list,
	// so when searching through the list of children, use that first invalid
	// index to find the child node.
	firstIndex := invalidIndices[0]
	node := n.children[firstIndex-parent]

	// If the child node doesn't exist in the list yet, fail fast by
	// returning, so we can construct and insert the proper inverted matrix.
	if node == nil {
		return nil
	}

	// If there's more than one invalid index left in the list we should
	// keep searching recursively.
	if len(invalidIndices) > 1 {
		// Search recursively on the child node by passing in the invalid indices
		// with the first index popped off the front.  Also the parent index to
		// pass down is the first index plus one.
		return node.getInvertedMatrix(invalidIndices[1:], firstIndex+1)
	}
	// If there aren't any more invalid indices to search, we've found our
	// node.  Return it, however keep in mind that the matrix could still be
	// nil because intermediary nodes in the tree are created sometimes with
	// their inversion matrices uninitialized.
	return node.matrix
}

func (n inversionNode) insertInvertedMatrix(invalidIndices []int, matrix matrix, shards, parent int) {
	// As above, get the child node to search next from the list of children.
	// The list of children starts relative to the parent index passed in
	// because the indices of invalid rows is sorted (by default).  As we
	// search recursively, the first invalid index gets popped off the list,
	// so when searching through the list of children, use that first invalid
	// index to find the child node.
	firstIndex := invalidIndices[0]
	node := n.children[firstIndex-parent]

	// If the child node doesn't exist in the list yet, create a new
	// node because we have the writer lock and add it to the list
	// of children.
	if node == nil {
		// Make the length of the list of children equal to the number
		// of shards minus the first invalid index because the list of
		// invalid indices is sorted, so only this length of errors
		// are possible in the tree.
		node = &inversionNode{
			children: make([]*inversionNode, shards-firstIndex),
		}
		// Insert the new node into the tree at the first index relative
		// to the parent index that was given in this recursive call.
		n.children[firstIndex-parent] = node
	}

	// If there's more than one invalid index left in the list we should
	// keep searching recursively in order to find the node to add our
	// matrix.
	if len(invalidIndices) > 1 {
		// As above, search recursively on the child node by passing in
		// the invalid indices with the first index popped off the front.
		// Also the total number of shards and parent index are passed down
		// which is equal to the first index plus one.
		node.insertInvertedMatrix(invalidIndices[1:], matrix, shards, firstIndex+1)
	} else {
		// If there aren't any more invalid indices to search, we've found our
		// node.  Cache the inverted matrix in this node.
		node.matrix = matrix
	}
}
