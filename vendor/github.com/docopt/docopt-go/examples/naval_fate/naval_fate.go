package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Naval Fate.

Usage:
  naval_fate ship new <name>...
  naval_fate ship <name> move <x> <y> [--speed=<kn>]
  naval_fate ship shoot <x> <y>
  naval_fate mine (set|remove) <x> <y> [--moored|--drifting]
  naval_fate -h | --help
  naval_fate --version

Options:
  -h --help     Show this screen.
  --version     Show version.
  --speed=<kn>  Speed in knots [default: 10].
  --moored      Moored (anchored) mine.
  --drifting    Drifting mine.`

	arguments, _ := docopt.Parse(usage, nil, true, "Naval Fate 2.0", false)
	fmt.Println(arguments)
}
