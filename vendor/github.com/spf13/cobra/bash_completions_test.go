package cobra

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func checkOmit(t *testing.T, found, unexpected string) {
	if strings.Contains(found, unexpected) {
		t.Errorf("Got: %q\nBut should not have!\n", unexpected)
	}
}

func check(t *testing.T, found, expected string) {
	if !strings.Contains(found, expected) {
		t.Errorf("Expecting to contain: \n %q\nGot:\n %q\n", expected, found)
	}
}

func checkRegex(t *testing.T, found, pattern string) {
	matched, err := regexp.MatchString(pattern, found)
	if err != nil {
		t.Errorf("Error thrown performing MatchString: \n %s\n", err)
	}
	if !matched {
		t.Errorf("Expecting to match: \n %q\nGot:\n %q\n", pattern, found)
	}
}

func runShellCheck(s string) error {
	excluded := []string{
		"SC2034", // PREFIX appears unused. Verify it or export it.
	}
	cmd := exec.Command("shellcheck", "-s", "bash", "-", "-e", strings.Join(excluded, ","))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		stdin.Write([]byte(s))
		stdin.Close()
	}()

	return cmd.Run()
}

// World worst custom function, just keep telling you to enter hello!
const bashCompletionFunc = `__custom_func() {
	COMPREPLY=( "hello" )
}
`

func TestBashCompletions(t *testing.T) {
	rootCmd := &Command{
		Use:                    "root",
		ArgAliases:             []string{"pods", "nodes", "services", "replicationcontrollers", "po", "no", "svc", "rc"},
		ValidArgs:              []string{"pod", "node", "service", "replicationcontroller"},
		BashCompletionFunction: bashCompletionFunc,
		Run: emptyRun,
	}
	rootCmd.Flags().IntP("introot", "i", -1, "help message for flag introot")
	rootCmd.MarkFlagRequired("introot")

	// Filename.
	rootCmd.Flags().String("filename", "", "Enter a filename")
	rootCmd.MarkFlagFilename("filename", "json", "yaml", "yml")

	// Persistent filename.
	rootCmd.PersistentFlags().String("persistent-filename", "", "Enter a filename")
	rootCmd.MarkPersistentFlagFilename("persistent-filename")
	rootCmd.MarkPersistentFlagRequired("persistent-filename")

	// Filename extensions.
	rootCmd.Flags().String("filename-ext", "", "Enter a filename (extension limited)")
	rootCmd.MarkFlagFilename("filename-ext")
	rootCmd.Flags().String("custom", "", "Enter a filename (extension limited)")
	rootCmd.MarkFlagCustom("custom", "__complete_custom")

	// Subdirectories in a given directory.
	rootCmd.Flags().String("theme", "", "theme to use (located in /themes/THEMENAME/)")
	rootCmd.Flags().SetAnnotation("theme", BashCompSubdirsInDir, []string{"themes"})

	echoCmd := &Command{
		Use:     "echo [string to echo]",
		Aliases: []string{"say"},
		Short:   "Echo anything to the screen",
		Long:    "an utterly useless command for testing.",
		Example: "Just run cobra-test echo",
		Run:     emptyRun,
	}

	echoCmd.Flags().String("filename", "", "Enter a filename")
	echoCmd.MarkFlagFilename("filename", "json", "yaml", "yml")
	echoCmd.Flags().String("config", "", "config to use (located in /config/PROFILE/)")
	echoCmd.Flags().SetAnnotation("config", BashCompSubdirsInDir, []string{"config"})

	printCmd := &Command{
		Use:   "print [string to print]",
		Args:  MinimumNArgs(1),
		Short: "Print anything to the screen",
		Long:  "an absolutely utterly useless command for testing.",
		Run:   emptyRun,
	}

	deprecatedCmd := &Command{
		Use:        "deprecated [can't do anything here]",
		Args:       NoArgs,
		Short:      "A command which is deprecated",
		Long:       "an absolutely utterly useless command for testing deprecation!.",
		Deprecated: "Please use echo instead",
		Run:        emptyRun,
	}

	colonCmd := &Command{
		Use: "cmd:colon",
		Run: emptyRun,
	}

	timesCmd := &Command{
		Use:        "times [# times] [string to echo]",
		SuggestFor: []string{"counts"},
		Args:       OnlyValidArgs,
		ValidArgs:  []string{"one", "two", "three", "four"},
		Short:      "Echo anything to the screen more times",
		Long:       "a slightly useless command for testing.",
		Run:        emptyRun,
	}

	echoCmd.AddCommand(timesCmd)
	rootCmd.AddCommand(echoCmd, printCmd, deprecatedCmd, colonCmd)

	buf := new(bytes.Buffer)
	rootCmd.GenBashCompletion(buf)
	output := buf.String()

	check(t, output, "_root")
	check(t, output, "_root_echo")
	check(t, output, "_root_echo_times")
	check(t, output, "_root_print")
	check(t, output, "_root_cmd__colon")

	// check for required flags
	check(t, output, `must_have_one_flag+=("--introot=")`)
	check(t, output, `must_have_one_flag+=("--persistent-filename=")`)
	// check for custom completion function
	check(t, output, `COMPREPLY=( "hello" )`)
	// check for required nouns
	check(t, output, `must_have_one_noun+=("pod")`)
	// check for noun aliases
	check(t, output, `noun_aliases+=("pods")`)
	check(t, output, `noun_aliases+=("rc")`)
	checkOmit(t, output, `must_have_one_noun+=("pods")`)
	// check for filename extension flags
	check(t, output, `flags_completion+=("_filedir")`)
	// check for filename extension flags
	check(t, output, `must_have_one_noun+=("three")`)
	// check for filename extension flags
	check(t, output, fmt.Sprintf(`flags_completion+=("__%s_handle_filename_extension_flag json|yaml|yml")`, rootCmd.Name()))
	// check for filename extension flags in a subcommand
	checkRegex(t, output, fmt.Sprintf(`_root_echo\(\)\n{[^}]*flags_completion\+=\("__%s_handle_filename_extension_flag json\|yaml\|yml"\)`, rootCmd.Name()))
	// check for custom flags
	check(t, output, `flags_completion+=("__complete_custom")`)
	// check for subdirs_in_dir flags
	check(t, output, fmt.Sprintf(`flags_completion+=("__%s_handle_subdirs_in_dir_flag themes")`, rootCmd.Name()))
	// check for subdirs_in_dir flags in a subcommand
	checkRegex(t, output, fmt.Sprintf(`_root_echo\(\)\n{[^}]*flags_completion\+=\("__%s_handle_subdirs_in_dir_flag config"\)`, rootCmd.Name()))

	checkOmit(t, output, deprecatedCmd.Name())

	// If available, run shellcheck against the script.
	if err := exec.Command("which", "shellcheck").Run(); err != nil {
		return
	}
	if err := runShellCheck(output); err != nil {
		t.Fatalf("shellcheck failed: %v", err)
	}
}

func TestBashCompletionHiddenFlag(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	const flagName = "hiddenFlag"
	c.Flags().Bool(flagName, false, "")
	c.Flags().MarkHidden(flagName)

	buf := new(bytes.Buffer)
	c.GenBashCompletion(buf)
	output := buf.String()

	if strings.Contains(output, flagName) {
		t.Errorf("Expected completion to not include %q flag: Got %v", flagName, output)
	}
}

func TestBashCompletionDeprecatedFlag(t *testing.T) {
	c := &Command{Use: "c", Run: emptyRun}

	const flagName = "deprecated-flag"
	c.Flags().Bool(flagName, false, "")
	c.Flags().MarkDeprecated(flagName, "use --not-deprecated instead")

	buf := new(bytes.Buffer)
	c.GenBashCompletion(buf)
	output := buf.String()

	if strings.Contains(output, flagName) {
		t.Errorf("expected completion to not include %q flag: Got %v", flagName, output)
	}
}
