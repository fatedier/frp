/**
 * Unit tests for Matrix
 *
 * Copyright 2015, Klaus Post
 * Copyright 2015, Backblaze, Inc.  All rights reserved.
 */

package reedsolomon

import (
	"testing"
)

// TestNewMatrix - Tests validate the result for invalid input and the allocations made by newMatrix method.
func TestNewMatrix(t *testing.T) {
	testCases := []struct {
		rows    int
		columns int

		// flag to indicate whether the test should pass.
		shouldPass     bool
		expectedResult matrix
		expectedErr    error
	}{
		// Test case - 1.
		// Test case with a negative row size.
		{-1, 10, false, nil, errInvalidRowSize},
		// Test case - 2.
		// Test case with a negative column size.
		{10, -1, false, nil, errInvalidColSize},
		// Test case - 3.
		// Test case with negative value for both row and column size.
		{-1, -1, false, nil, errInvalidRowSize},
		// Test case - 4.
		// Test case with 0 value for row size.
		{0, 10, false, nil, errInvalidRowSize},
		// Test case - 5.
		// Test case with 0 value for column size.
		{-1, 0, false, nil, errInvalidRowSize},
		// Test case - 6.
		// Test case with 0 value for both row and column size.
		{0, 0, false, nil, errInvalidRowSize},
	}
	for i, testCase := range testCases {
		actualResult, actualErr := newMatrix(testCase.rows, testCase.columns)
		if actualErr != nil && testCase.shouldPass {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, actualErr.Error())
		}
		if actualErr == nil && !testCase.shouldPass {
			t.Errorf("Test %d: Expected to fail with <ERROR> \"%s\", but passed instead.", i+1, testCase.expectedErr)
		}
		// Failed as expected, but does it fail for the expected reason.
		if actualErr != nil && !testCase.shouldPass {
			if testCase.expectedErr != actualErr {
				t.Errorf("Test %d: Expected to fail with error \"%s\", but instead failed with error \"%s\" instead.", i+1, testCase.expectedErr, actualErr)
			}
		}
		// Test passes as expected, but the output values
		// are verified for correctness here.
		if actualErr == nil && testCase.shouldPass {
			if testCase.rows != len(actualResult) {
				// End the tests here if the the size doesn't match number of rows.
				t.Fatalf("Test %d: Expected the size of the row of the new matrix to be `%d`, but instead found `%d`", i+1, testCase.rows, len(actualResult))
			}
			// Iterating over each row and validating the size of the column.
			for j, row := range actualResult {
				// If the row check passes, verify the size of each columns.
				if testCase.columns != len(row) {
					t.Errorf("Test %d: Row %d: Expected the size of the column of the new matrix to be `%d`, but instead found `%d`", i+1, j+1, testCase.columns, len(row))
				}
			}
		}
	}
}

// TestMatrixIdentity - validates the method for returning identity matrix of given size.
func TestMatrixIdentity(t *testing.T) {
	m, err := identityMatrix(3)
	if err != nil {
		t.Fatal(err)
	}
	str := m.String()
	expect := "[[1, 0, 0], [0, 1, 0], [0, 0, 1]]"
	if str != expect {
		t.Fatal(str, "!=", expect)
	}
}

// Tests validate the output of matix multiplication method.
func TestMatrixMultiply(t *testing.T) {
	m1, err := newMatrixData(
		[][]byte{
			[]byte{1, 2},
			[]byte{3, 4},
		})
	if err != nil {
		t.Fatal(err)
	}

	m2, err := newMatrixData(
		[][]byte{
			[]byte{5, 6},
			[]byte{7, 8},
		})
	if err != nil {
		t.Fatal(err)
	}
	actual, err := m1.Multiply(m2)
	if err != nil {
		t.Fatal(err)
	}
	str := actual.String()
	expect := "[[11, 22], [19, 42]]"
	if str != expect {
		t.Fatal(str, "!=", expect)
	}
}

// Tests validate the output of the method with computes inverse of matrix.
func TestMatrixInverse(t *testing.T) {
	testCases := []struct {
		matrixData [][]byte
		// expected inverse matrix.
		expectedResult string
		// flag indicating whether the test should pass.
		shouldPass  bool
		expectedErr error
	}{
		// Test case - 1.
		// Test case validating inverse of the input Matrix.
		{
			// input data to construct the matrix.
			[][]byte{
				[]byte{56, 23, 98},
				[]byte{3, 100, 200},
				[]byte{45, 201, 123},
			},
			// expected Inverse matrix.
			"[[175, 133, 33], [130, 13, 245], [112, 35, 126]]",
			// test is expected to pass.
			true,
			nil,
		},
		// Test case - 2.
		// Test case validating inverse of the input Matrix.
		{
			// input data to contruct the matrix.
			[][]byte{
				[]byte{1, 0, 0, 0, 0},
				[]byte{0, 1, 0, 0, 0},
				[]byte{0, 0, 0, 1, 0},
				[]byte{0, 0, 0, 0, 1},
				[]byte{7, 7, 6, 6, 1},
			},
			// expectedInverse matrix.
			"[[1, 0, 0, 0, 0]," +
				" [0, 1, 0, 0, 0]," +
				" [123, 123, 1, 122, 122]," +
				" [0, 0, 1, 0, 0]," +
				" [0, 0, 0, 1, 0]]",
			// test is expected to pass.
			true,
			nil,
		},
		// Test case with a non-square matrix.
		// expected to fail with errNotSquare.
		{
			[][]byte{
				[]byte{56, 23},
				[]byte{3, 100},
				[]byte{45, 201},
			},
			"",
			false,
			errNotSquare,
		},
		// Test case with singular matrix.
		// expected to fail with error errSingular.
		{

			[][]byte{
				[]byte{4, 2},
				[]byte{12, 6},
			},
			"",
			false,
			errSingular,
		},
	}

	for i, testCase := range testCases {
		m, err := newMatrixData(testCase.matrixData)
		if err != nil {
			t.Fatalf("Test %d: Failed initializing new Matrix : %s", i+1, err)
		}
		actualResult, actualErr := m.Invert()
		if actualErr != nil && testCase.shouldPass {
			t.Errorf("Test %d: Expected to pass, but failed with: <ERROR> %s", i+1, actualErr.Error())
		}
		if actualErr == nil && !testCase.shouldPass {
			t.Errorf("Test %d: Expected to fail with <ERROR> \"%s\", but passed instead.", i+1, testCase.expectedErr)
		}
		// Failed as expected, but does it fail for the expected reason.
		if actualErr != nil && !testCase.shouldPass {
			if testCase.expectedErr != actualErr {
				t.Errorf("Test %d: Expected to fail with error \"%s\", but instead failed with error \"%s\" instead.", i+1, testCase.expectedErr, actualErr)
			}
		}
		// Test passes as expected, but the output values
		// are verified for correctness here.
		if actualErr == nil && testCase.shouldPass {
			if testCase.expectedResult != actualResult.String() {
				t.Errorf("Test %d: The inverse matrix doesnt't match the expected result", i+1)
			}
		}
	}
}
