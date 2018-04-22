package cobra

import (
	"bytes"
	"strings"
	"testing"
)

func TestZshCompletion(t *testing.T) {
	tcs := []struct {
		name                string
		root                *Command
		expectedExpressions []string
	}{
		{
			name:                "trivial",
			root:                &Command{Use: "trivialapp"},
			expectedExpressions: []string{"#compdef trivial"},
		},
		{
			name: "linear",
			root: func() *Command {
				r := &Command{Use: "linear"}

				sub1 := &Command{Use: "sub1"}
				r.AddCommand(sub1)

				sub2 := &Command{Use: "sub2"}
				sub1.AddCommand(sub2)

				sub3 := &Command{Use: "sub3"}
				sub2.AddCommand(sub3)
				return r
			}(),
			expectedExpressions: []string{"sub1", "sub2", "sub3"},
		},
		{
			name: "flat",
			root: func() *Command {
				r := &Command{Use: "flat"}
				r.AddCommand(&Command{Use: "c1"})
				r.AddCommand(&Command{Use: "c2"})
				return r
			}(),
			expectedExpressions: []string{"(c1 c2)"},
		},
		{
			name: "tree",
			root: func() *Command {
				r := &Command{Use: "tree"}

				sub1 := &Command{Use: "sub1"}
				r.AddCommand(sub1)

				sub11 := &Command{Use: "sub11"}
				sub12 := &Command{Use: "sub12"}

				sub1.AddCommand(sub11)
				sub1.AddCommand(sub12)

				sub2 := &Command{Use: "sub2"}
				r.AddCommand(sub2)

				sub21 := &Command{Use: "sub21"}
				sub22 := &Command{Use: "sub22"}

				sub2.AddCommand(sub21)
				sub2.AddCommand(sub22)

				return r
			}(),
			expectedExpressions: []string{"(sub11 sub12)", "(sub21 sub22)"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			tc.root.GenZshCompletion(buf)
			output := buf.String()

			for _, expectedExpression := range tc.expectedExpressions {
				if !strings.Contains(output, expectedExpression) {
					t.Errorf("Expected completion to contain %q somewhere; got %q", expectedExpression, output)
				}
			}
		})
	}
}
