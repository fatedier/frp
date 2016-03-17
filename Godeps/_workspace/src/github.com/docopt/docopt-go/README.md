docopt-go
=========

[![Build Status](https://travis-ci.org/docopt/docopt.go.svg?branch=master)](https://travis-ci.org/docopt/docopt.go)
[![Coverage Status](https://coveralls.io/repos/docopt/docopt.go/badge.png)](https://coveralls.io/r/docopt/docopt.go)
[![GoDoc](https://godoc.org/github.com/docopt/docopt.go?status.png)](https://godoc.org/github.com/docopt/docopt.go)

An implementation of [docopt](http://docopt.org/) in the
[Go](http://golang.org/) programming language.

**docopt** helps you create *beautiful* command-line interfaces easily:

```go
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
```

**docopt** parses command-line arguments based on a help message. Don't
write parser code: a good help message already has all the necessary
information in it.

## Installation

⚠ Use the alias “docopt-go”. To use docopt in your Go code:

```go
import "github.com/docopt/docopt-go"
```

To install docopt according to your `$GOPATH`:

```console
$ go get github.com/docopt/docopt-go
```

## API

```go
func Parse(doc string, argv []string, help bool, version string,
    optionsFirst bool, exit ...bool) (map[string]interface{}, error)
```
Parse `argv` based on the command-line interface described in `doc`.

Given a conventional command-line help message, docopt creates a parser and
processes the arguments. See
https://github.com/docopt/docopt#help-message-format for a description of the
help message format. If `argv` is `nil`, `os.Args[1:]` is used.

docopt returns a map of option names to the values parsed from `argv`, and an
error or `nil`.

More documentation for docopt is available at
[GoDoc.org](https://godoc.org/github.com/docopt/docopt.go).

## Testing

All tests from the Python version are implemented and passing
at [Travis CI](https://travis-ci.org/docopt/docopt.go). New
language-agnostic tests have been added
to [test_golang.docopt](test_golang.docopt).

To run tests for docopt-go, use `go test`.
