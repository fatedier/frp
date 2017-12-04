package git

import (
	"fmt"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `usage: git remote [-v | --verbose]
       git remote add [-t <branch>] [-m <master>] [-f] [--mirror] <name> <url>
       git remote rename <old> <new>
       git remote rm <name>
       git remote set-head <name> (-a | -d | <branch>)
       git remote [-v | --verbose] show [-n] <name>
       git remote prune [-n | --dry-run] <name>
       git remote [-v | --verbose] update [-p | --prune] [(<group> | <remote>)...]
       git remote set-branches <name> [--add] <branch>...
       git remote set-url <name> <newurl> [<oldurl>]
       git remote set-url --add <name> <newurl>
       git remote set-url --delete <name> <url>

options:
    -v, --verbose         be verbose; must be placed before a subcommand
`

	args, _ := docopt.Parse(usage, nil, true, "", false)
	fmt.Println(args)
}
