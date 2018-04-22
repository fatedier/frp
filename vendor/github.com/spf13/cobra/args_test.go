package cobra

import (
	"strings"
	"testing"
)

func TestNoArgs(t *testing.T) {
	c := &Command{Use: "c", Args: NoArgs, Run: emptyRun}

	output, err := executeCommand(c)
	if output != "" {
		t.Errorf("Unexpected string: %v", output)
	}
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestNoArgsWithArgs(t *testing.T) {
	c := &Command{Use: "c", Args: NoArgs, Run: emptyRun}

	_, err := executeCommand(c, "illegal")
	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := `unknown command "illegal" for "c"`
	if got != expected {
		t.Errorf("Expected: %q, got: %q", expected, got)
	}
}

func TestOnlyValidArgs(t *testing.T) {
	c := &Command{
		Use:       "c",
		Args:      OnlyValidArgs,
		ValidArgs: []string{"one", "two"},
		Run:       emptyRun,
	}

	output, err := executeCommand(c, "one", "two")
	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestOnlyValidArgsWithInvalidArgs(t *testing.T) {
	c := &Command{
		Use:       "c",
		Args:      OnlyValidArgs,
		ValidArgs: []string{"one", "two"},
		Run:       emptyRun,
	}

	_, err := executeCommand(c, "three")
	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := `invalid argument "three" for "c"`
	if got != expected {
		t.Errorf("Expected: %q, got: %q", expected, got)
	}
}

func TestArbitraryArgs(t *testing.T) {
	c := &Command{Use: "c", Args: ArbitraryArgs, Run: emptyRun}
	output, err := executeCommand(c, "a", "b")
	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMinimumNArgs(t *testing.T) {
	c := &Command{Use: "c", Args: MinimumNArgs(2), Run: emptyRun}
	output, err := executeCommand(c, "a", "b", "c")
	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMinimumNArgsWithLessArgs(t *testing.T) {
	c := &Command{Use: "c", Args: MinimumNArgs(2), Run: emptyRun}
	_, err := executeCommand(c, "a")

	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := "requires at least 2 arg(s), only received 1"
	if got != expected {
		t.Fatalf("Expected %q, got %q", expected, got)
	}
}

func TestMaximumNArgs(t *testing.T) {
	c := &Command{Use: "c", Args: MaximumNArgs(3), Run: emptyRun}
	output, err := executeCommand(c, "a", "b")
	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMaximumNArgsWithMoreArgs(t *testing.T) {
	c := &Command{Use: "c", Args: MaximumNArgs(2), Run: emptyRun}
	_, err := executeCommand(c, "a", "b", "c")

	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := "accepts at most 2 arg(s), received 3"
	if got != expected {
		t.Fatalf("Expected %q, got %q", expected, got)
	}
}

func TestExactArgs(t *testing.T) {
	c := &Command{Use: "c", Args: ExactArgs(3), Run: emptyRun}
	output, err := executeCommand(c, "a", "b", "c")
	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestExactArgsWithInvalidCount(t *testing.T) {
	c := &Command{Use: "c", Args: ExactArgs(2), Run: emptyRun}
	_, err := executeCommand(c, "a", "b", "c")

	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := "accepts 2 arg(s), received 3"
	if got != expected {
		t.Fatalf("Expected %q, got %q", expected, got)
	}
}

func TestRangeArgs(t *testing.T) {
	c := &Command{Use: "c", Args: RangeArgs(2, 4), Run: emptyRun}
	output, err := executeCommand(c, "a", "b", "c")
	if output != "" {
		t.Errorf("Unexpected output: %v", output)
	}
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRangeArgsWithInvalidCount(t *testing.T) {
	c := &Command{Use: "c", Args: RangeArgs(2, 4), Run: emptyRun}
	_, err := executeCommand(c, "a")

	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := "accepts between 2 and 4 arg(s), received 1"
	if got != expected {
		t.Fatalf("Expected %q, got %q", expected, got)
	}
}

func TestRootTakesNoArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Run: emptyRun}
	childCmd := &Command{Use: "child", Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "illegal", "args")
	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := `unknown command "illegal" for "root"`
	if !strings.Contains(got, expected) {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRootTakesArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Args: ArbitraryArgs, Run: emptyRun}
	childCmd := &Command{Use: "child", Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "legal", "args")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestChildTakesNoArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Run: emptyRun}
	childCmd := &Command{Use: "child", Args: NoArgs, Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "child", "illegal", "args")
	if err == nil {
		t.Fatal("Expected an error")
	}

	got := err.Error()
	expected := `unknown command "illegal" for "root child"`
	if !strings.Contains(got, expected) {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestChildTakesArgs(t *testing.T) {
	rootCmd := &Command{Use: "root", Run: emptyRun}
	childCmd := &Command{Use: "child", Args: ArbitraryArgs, Run: emptyRun}
	rootCmd.AddCommand(childCmd)

	_, err := executeCommand(rootCmd, "child", "legal", "args")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
