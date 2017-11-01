/**
 * Unit tests for inversion tree.
 *
 * Copyright 2016, Peter Collins
 */

package reedsolomon

import (
	"testing"
)

func TestNewInversionTree(t *testing.T) {
	tree := newInversionTree(3, 2)

	children := len(tree.root.children)
	if children != 5 {
		t.Fatal("Root node children list length", children, "!=", 5)
	}

	str := tree.root.matrix.String()
	expect := "[[1, 0, 0], [0, 1, 0], [0, 0, 1]]"
	if str != expect {
		t.Fatal(str, "!=", expect)
	}
}

func TestGetInvertedMatrix(t *testing.T) {
	tree := newInversionTree(3, 2)

	matrix := tree.GetInvertedMatrix([]int{})
	str := matrix.String()
	expect := "[[1, 0, 0], [0, 1, 0], [0, 0, 1]]"
	if str != expect {
		t.Fatal(str, "!=", expect)
	}

	matrix = tree.GetInvertedMatrix([]int{1})
	if matrix != nil {
		t.Fatal(matrix, "!= nil")
	}

	matrix = tree.GetInvertedMatrix([]int{1, 2})
	if matrix != nil {
		t.Fatal(matrix, "!= nil")
	}

	matrix, err := newMatrix(3, 3)
	if err != nil {
		t.Fatalf("Failed initializing new Matrix : %s", err)
	}
	err = tree.InsertInvertedMatrix([]int{1}, matrix, 5)
	if err != nil {
		t.Fatalf("Failed inserting new Matrix : %s", err)
	}

	cachedMatrix := tree.GetInvertedMatrix([]int{1})
	if cachedMatrix == nil {
		t.Fatal(cachedMatrix, "== nil")
	}
	if matrix.String() != cachedMatrix.String() {
		t.Fatal(matrix.String(), "!=", cachedMatrix.String())
	}
}

func TestInsertInvertedMatrix(t *testing.T) {
	tree := newInversionTree(3, 2)

	matrix, err := newMatrix(3, 3)
	if err != nil {
		t.Fatalf("Failed initializing new Matrix : %s", err)
	}
	err = tree.InsertInvertedMatrix([]int{1}, matrix, 5)
	if err != nil {
		t.Fatalf("Failed inserting new Matrix : %s", err)
	}

	err = tree.InsertInvertedMatrix([]int{}, matrix, 5)
	if err == nil {
		t.Fatal("Should have failed inserting the root node matrix", matrix)
	}

	matrix, err = newMatrix(3, 2)
	if err != nil {
		t.Fatalf("Failed initializing new Matrix : %s", err)
	}
	err = tree.InsertInvertedMatrix([]int{2}, matrix, 5)
	if err == nil {
		t.Fatal("Should have failed inserting a non-square matrix", matrix)
	}

	matrix, err = newMatrix(3, 3)
	if err != nil {
		t.Fatalf("Failed initializing new Matrix : %s", err)
	}
	err = tree.InsertInvertedMatrix([]int{0, 1}, matrix, 5)
	if err != nil {
		t.Fatalf("Failed inserting new Matrix : %s", err)
	}
}

func TestDoubleInsertInvertedMatrix(t *testing.T) {
	tree := newInversionTree(3, 2)

	matrix, err := newMatrix(3, 3)
	if err != nil {
		t.Fatalf("Failed initializing new Matrix : %s", err)
	}
	err = tree.InsertInvertedMatrix([]int{1}, matrix, 5)
	if err != nil {
		t.Fatalf("Failed inserting new Matrix : %s", err)
	}
	err = tree.InsertInvertedMatrix([]int{1}, matrix, 5)
	if err != nil {
		t.Fatalf("Failed inserting new Matrix : %s", err)
	}

	cachedMatrix := tree.GetInvertedMatrix([]int{1})
	if cachedMatrix == nil {
		t.Fatal(cachedMatrix, "== nil")
	}
	if matrix.String() != cachedMatrix.String() {
		t.Fatal(matrix.String(), "!=", cachedMatrix.String())
	}
}
