package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"os"
	"os/exec"
)

func main() {
	usage := `usage: git [--version] [--exec-path=<path>] [--html-path]
           [-p|--paginate|--no-pager] [--no-replace-objects]
           [--bare] [--git-dir=<path>] [--work-tree=<path>]
           [-c <name>=<value>] [--help]
           <command> [<args>...]

options:
   -c <name=value>
   -h, --help
   -p, --paginate

The most commonly used git commands are:
   add        Add file contents to the index
   branch     List, create, or delete branches
   checkout   Checkout a branch or paths to the working tree
   clone      Clone a repository into a new directory
   commit     Record changes to the repository
   push       Update remote refs along with associated objects
   remote     Manage set of tracked repositories

See 'git help <command>' for more information on a specific command.
`
	args, _ := docopt.Parse(usage, nil, true, "git version 1.7.4.4", true)

	fmt.Println("global arguments:")
	fmt.Println(args)

	fmt.Println("command arguments:")
	cmd := args["<command>"].(string)
	cmdArgs := args["<args>"].([]string)

	err := runCommand(cmd, cmdArgs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func goRun(scriptName string, args []string) (err error) {
	cmdArgs := make([]string, 2)
	cmdArgs[0] = "run"
	cmdArgs[1] = scriptName
	cmdArgs = append(cmdArgs, args...)
	osCmd := exec.Command("go", cmdArgs...)
	var out []byte
	out, err = osCmd.Output()
	fmt.Println(string(out))
	if err != nil {
		return
	}
	return
}

func runCommand(cmd string, args []string) (err error) {
	argv := make([]string, 1)
	argv[0] = cmd
	argv = append(argv, args...)
	switch cmd {
	case "add":
		// subcommand is a function call
		return cmdAdd(argv)
	case "branch":
		// subcommand is a script
		return goRun("branch/git_branch.go", argv)
	case "checkout", "clone", "commit", "push", "remote":
		// subcommand is a script
		scriptName := fmt.Sprintf("%s/git_%s.go", cmd, cmd)
		return goRun(scriptName, argv)
	case "help", "":
		return goRun("git.go", []string{"git_add.go", "--help"})
	}

	return fmt.Errorf("%s is not a git command. See 'git help'", cmd)
}

func cmdAdd(argv []string) (err error) {
	usage := `usage: git add [options] [--] [<filepattern>...]

options:
	-h, --help
	-n, --dry-run        dry run
	-v, --verbose        be verbose
	-i, --interactive    interactive picking
	-p, --patch          select hunks interactively
	-e, --edit           edit current diff and apply
	-f, --force          allow adding otherwise ignored files
	-u, --update         update tracked files
	-N, --intent-to-add  record only the fact that the path will be added later
	-A, --all            add all, noticing removal of tracked files
	--refresh            don't add, only refresh the index
	--ignore-errors      just skip files which cannot be added because of errors
	--ignore-missing     check if - even missing - files are ignored in dry run
`

	args, _ := docopt.Parse(usage, nil, true, "", false)
	fmt.Println(args)
	return
}
