package git

import (
	"fmt"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `usage: git push [options] [<repository> [<refspec>...]]

options:
    -h, --help
    -v, --verbose         be more verbose
    -q, --quiet           be more quiet
    --repo <repository>   repository
    --all                 push all refs
    --mirror              mirror all refs
    --delete              delete refs
    --tags                push tags (can't be used with --all or --mirror)
    -n, --dry-run         dry run
    --porcelain           machine-readable output
    -f, --force           force updates
    --thin                use thin pack
    --receive-pack <receive-pack>
                          receive pack program
    --exec <receive-pack>
                          receive pack program
    -u, --set-upstream    set upstream for git pull/status
    --progress            force progress reporting
`

	args, _ := docopt.Parse(usage, nil, true, "", false)
	fmt.Println(args)
}
