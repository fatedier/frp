package cmd

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
)

var update = flag.Bool("update", false, "update .golden files")

func init() {
	// Mute commands.
	addCmd.SetOutput(new(bytes.Buffer))
	initCmd.SetOutput(new(bytes.Buffer))
}

// compareFiles compares the content of files with pathA and pathB.
// If contents are equal, it returns nil.
// If not, it returns which files are not equal
// and diff (if system has diff command) between these files.
func compareFiles(pathA, pathB string) error {
	contentA, err := ioutil.ReadFile(pathA)
	if err != nil {
		return err
	}
	contentB, err := ioutil.ReadFile(pathB)
	if err != nil {
		return err
	}
	if !bytes.Equal(contentA, contentB) {
		output := new(bytes.Buffer)
		output.WriteString(fmt.Sprintf("%q and %q are not equal!\n\n", pathA, pathB))

		diffPath, err := exec.LookPath("diff")
		if err != nil {
			// Don't execute diff if it can't be found.
			return nil
		}
		diffCmd := exec.Command(diffPath, "-u", pathA, pathB)
		diffCmd.Stdout = output
		diffCmd.Stderr = output

		output.WriteString("$ diff -u " + pathA + " " + pathB + "\n")
		if err := diffCmd.Run(); err != nil {
			output.WriteString("\n" + err.Error())
		}
		return errors.New(output.String())
	}
	return nil
}

// checkLackFiles checks if all elements of expected are in got.
func checkLackFiles(expected, got []string) error {
	lacks := make([]string, 0, len(expected))
	for _, ev := range expected {
		if !stringInStringSlice(ev, got) {
			lacks = append(lacks, ev)
		}
	}
	if len(lacks) > 0 {
		return fmt.Errorf("Lack %v file(s): %v", len(lacks), lacks)
	}
	return nil
}

// stringInStringSlice checks if s is an element of slice.
func stringInStringSlice(s string, slice []string) bool {
	for _, v := range slice {
		if s == v {
			return true
		}
	}
	return false
}
