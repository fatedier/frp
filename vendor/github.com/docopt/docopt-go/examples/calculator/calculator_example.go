package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `Not a serious example.

Usage:
  calculator_example <value> ( ( + | - | * | / ) <value> )...
  calculator_example <function> <value> [( , <value> )]...
  calculator_example (-h | --help)

Examples:
  calculator_example 1 + 2 + 3 + 4 + 5
  calculator_example 1 + 2 '*' 3 / 4 - 5    # note quotes around '*'
  calculator_example sum 10 , 20 , 30 , 40

Options:
  -h, --help
`
	arguments, _ := docopt.Parse(usage, nil, true, "", false)
	fmt.Println(arguments)
}
