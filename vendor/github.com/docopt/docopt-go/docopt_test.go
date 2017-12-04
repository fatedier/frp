/*
Based of off docopt.py: https://github.com/docopt/docopt

Licensed under terms of MIT license (see LICENSE-MIT)
Copyright (c) 2013 Keith Batten, kbatten@gmail.com
*/

package docopt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestPatternFlat(t *testing.T) {
	q := patternList{
		newArgument("N", nil),
		newOption("-a", "", 0, false),
		newArgument("M", nil)}
	p, err := newRequired(
		newOneOrMore(newArgument("N", nil)),
		newOption("-a", "", 0, false),
		newArgument("M", nil)).flat(patternDefault)
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	q = patternList{newOptionsShortcut()}
	p, err = newRequired(
		newOptional(newOptionsShortcut()),
		newOptional(newOption("-a", "", 0, false))).flat(patternOptionSSHORTCUT)
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}
	return
}

func TestOption(t *testing.T) {
	if !parseOption("-h").eq(newOption("-h", "", 0, false)) {
		t.Fail()
	}
	if !parseOption("--help").eq(newOption("", "--help", 0, false)) {
		t.Fail()
	}
	if !parseOption("-h --help").eq(newOption("-h", "--help", 0, false)) {
		t.Fail()
	}
	if !parseOption("-h, --help").eq(newOption("-h", "--help", 0, false)) {
		t.Fail()
	}

	if !parseOption("-h TOPIC").eq(newOption("-h", "", 1, false)) {
		t.Fail()
	}
	if !parseOption("--help TOPIC").eq(newOption("", "--help", 1, false)) {
		t.Fail()
	}
	if !parseOption("-h TOPIC --help TOPIC").eq(newOption("-h", "--help", 1, false)) {
		t.Fail()
	}
	if !parseOption("-h TOPIC, --help TOPIC").eq(newOption("-h", "--help", 1, false)) {
		t.Fail()
	}
	if !parseOption("-h TOPIC, --help=TOPIC").eq(newOption("-h", "--help", 1, false)) {
		t.Fail()
	}

	if !parseOption("-h  Description...").eq(newOption("-h", "", 0, false)) {
		t.Fail()
	}
	if !parseOption("-h --help  Description...").eq(newOption("-h", "--help", 0, false)) {
		t.Fail()
	}
	if !parseOption("-h TOPIC  Description...").eq(newOption("-h", "", 1, false)) {
		t.Fail()
	}

	if !parseOption("    -h").eq(newOption("-h", "", 0, false)) {
		t.Fail()
	}

	if !parseOption("-h TOPIC  Description... [default: 2]").eq(newOption("-h", "", 1, "2")) {
		t.Fail()
	}
	if !parseOption("-h TOPIC  Descripton... [default: topic-1]").eq(newOption("-h", "", 1, "topic-1")) {
		t.Fail()
	}
	if !parseOption("--help=TOPIC  ... [default: 3.14]").eq(newOption("", "--help", 1, "3.14")) {
		t.Fail()
	}
	if !parseOption("-h, --help=DIR  ... [default: ./]").eq(newOption("-h", "--help", 1, "./")) {
		t.Fail()
	}
	if !parseOption("-h TOPIC  Descripton... [dEfAuLt: 2]").eq(newOption("-h", "", 1, "2")) {
		t.Fail()
	}
	return
}

func TestOptionName(t *testing.T) {
	if newOption("-h", "", 0, false).name != "-h" {
		t.Fail()
	}
	if newOption("-h", "--help", 0, false).name != "--help" {
		t.Fail()
	}
	if newOption("", "--help", 0, false).name != "--help" {
		t.Fail()
	}
	return
}

func TestCommands(t *testing.T) {
	if v, err := Parse("Usage: prog add", []string{"add"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"add": true}) != true {
		t.Error(err)
	}
	if v, err := Parse("Usage: prog [add]", []string{}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"add": false}) != true {
		t.Error(err)
	}
	if v, err := Parse("Usage: prog [add]", []string{"add"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"add": true}) != true {
		t.Error(err)
	}
	if v, err := Parse("Usage: prog (add|rm)", []string{"add"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"add": true, "rm": false}) != true {
		t.Error(err)
	}
	if v, err := Parse("Usage: prog (add|rm)", []string{"rm"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"add": false, "rm": true}) != true {
		t.Error(err)
	}
	if v, err := Parse("Usage: prog a b", []string{"a", "b"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"a": true, "b": true}) != true {
		t.Error(err)
	}
	_, err := Parse("Usage: prog a b", []string{"b", "a"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
	return
}

func TestFormalUsage(t *testing.T) {
	doc := `
    Usage: prog [-hv] ARG
           prog N M

    prog is a program`
	usage := parseSection("usage:", doc)[0]
	if usage != "Usage: prog [-hv] ARG\n           prog N M" {
		t.FailNow()
	}
	formal, err := formalUsage(usage)
	if err != nil {
		t.Fatal(err)
	}
	if formal != "( [-hv] ARG ) | ( N M )" {
		t.Fail()
	}
	return
}

func TestParseArgv(t *testing.T) {
	o := patternList{
		newOption("-h", "", 0, false),
		newOption("-v", "--verbose", 0, false),
		newOption("-f", "--file", 1, false),
	}

	p, err := parseArgv(tokenListFromString(""), &o, false)
	q := patternList{}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	p, err = parseArgv(tokenListFromString("-h"), &o, false)
	q = patternList{newOption("-h", "", 0, true)}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	p, err = parseArgv(tokenListFromString("-h --verbose"), &o, false)
	q = patternList{
		newOption("-h", "", 0, true),
		newOption("-v", "--verbose", 0, true),
	}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	p, err = parseArgv(tokenListFromString("-h --file f.txt"), &o, false)
	q = patternList{
		newOption("-h", "", 0, true),
		newOption("-f", "--file", 1, "f.txt"),
	}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	p, err = parseArgv(tokenListFromString("-h --file f.txt arg"), &o, false)
	q = patternList{
		newOption("-h", "", 0, true),
		newOption("-f", "--file", 1, "f.txt"),
		newArgument("", "arg"),
	}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	p, err = parseArgv(tokenListFromString("-h --file f.txt arg arg2"), &o, false)
	q = patternList{
		newOption("-h", "", 0, true),
		newOption("-f", "--file", 1, "f.txt"),
		newArgument("", "arg"),
		newArgument("", "arg2"),
	}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}

	p, err = parseArgv(tokenListFromString("-h arg -- -v"), &o, false)
	q = patternList{
		newOption("-h", "", 0, true),
		newArgument("", "arg"),
		newArgument("", "--"),
		newArgument("", "-v"),
	}
	if reflect.DeepEqual(p, q) != true {
		t.Error(err)
	}
}

func TestParsePattern(t *testing.T) {
	o := patternList{
		newOption("-h", "", 0, false),
		newOption("-v", "--verbose", 0, false),
		newOption("-f", "--file", 1, false),
	}

	p, err := parsePattern("[ -h ]", &o)
	q := newRequired(newOptional(newOption("-h", "", 0, false)))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("[ ARG ... ]", &o)
	q = newRequired(newOptional(
		newOneOrMore(
			newArgument("ARG", nil))))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("[ -h | -v ]", &o)
	q = newRequired(
		newOptional(
			newEither(
				newOption("-h", "", 0, false),
				newOption("-v", "--verbose", 0, false))))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("( -h | -v [ --file <f> ] )", &o)
	q = newRequired(
		newRequired(
			newEither(
				newOption("-h", "", 0, false),
				newRequired(
					newOption("-v", "--verbose", 0, false),
					newOptional(
						newOption("-f", "--file", 1, nil))))))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("(-h|-v[--file=<f>]N...)", &o)
	q = newRequired(
		newRequired(
			newEither(
				newOption("-h", "", 0, false),
				newRequired(
					newOption("-v", "--verbose", 0, false),
					newOptional(
						newOption("-f", "--file", 1, nil)),
					newOneOrMore(
						newArgument("N", nil))))))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("(N [M | (K | L)] | O P)", &o)
	q = newRequired(
		newRequired(
			newEither(
				newRequired(
					newArgument("N", nil),
					newOptional(
						newEither(
							newArgument("M", nil),
							newRequired(
								newEither(
									newArgument("K", nil),
									newArgument("L", nil)))))),
				newRequired(
					newArgument("O", nil),
					newArgument("P", nil)))))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("[ -h ] [N]", &o)
	q = newRequired(
		newOptional(
			newOption("-h", "", 0, false)),
		newOptional(
			newArgument("N", nil)))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("[options]", &o)
	q = newRequired(
		newOptional(
			newOptionsShortcut()))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("[options] A", &o)
	q = newRequired(
		newOptional(
			newOptionsShortcut()),
		newArgument("A", nil))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("-v [options]", &o)
	q = newRequired(
		newOption("-v", "--verbose", 0, false),
		newOptional(
			newOptionsShortcut()))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("ADD", &o)
	q = newRequired(newArgument("ADD", nil))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("<add>", &o)
	q = newRequired(newArgument("<add>", nil))
	if p.eq(q) != true {
		t.Error(err)
	}

	p, err = parsePattern("add", &o)
	q = newRequired(newCommand("add", false))
	if p.eq(q) != true {
		t.Error(err)
	}
}

func TestOptionMatch(t *testing.T) {
	v, w, x := newOption("-a", "", 0, false).match(
		&patternList{newOption("-a", "", 0, true)}, nil)
	y := patternList{newOption("-a", "", 0, true)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newOption("-a", "", 0, false).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newOption("-a", "", 0, false).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}
	v, w, x = newOption("-a", "", 0, false).match(
		&patternList{newArgument("N", nil)}, nil)
	y = patternList{newArgument("N", nil)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newOption("-a", "", 0, false).match(
		&patternList{
			newOption("-x", "", 0, false),
			newOption("-a", "", 0, false),
			newArgument("N", nil)}, nil)
	y = patternList{
		newOption("-x", "", 0, false),
		newArgument("N", nil)}
	z := patternList{newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOption("-a", "", 0, false).match(
		&patternList{
			newOption("-a", "", 0, true),
			newOption("-a", "", 0, false)}, nil)
	y = patternList{newOption("-a", "", 0, false)}
	z = patternList{newOption("-a", "", 0, true)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
}

func TestArgumentMatch(t *testing.T) {
	v, w, x := newArgument("N", nil).match(
		&patternList{newArgument("N", 9)}, nil)
	y := patternList{newArgument("N", 9)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newArgument("N", nil).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newArgument("N", nil).match(
		&patternList{newOption("-x", "", 0, false),
			newOption("-a", "", 0, false),
			newArgument("", 5)}, nil)
	y = patternList{newOption("-x", "", 0, false),
		newOption("-a", "", 0, false)}
	z := patternList{newArgument("N", 5)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newArgument("N", nil).match(
		&patternList{newArgument("", 9),
			newArgument("", 0)}, nil)
	y = patternList{newArgument("", 0)}
	z = patternList{newArgument("N", 9)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
}

func TestCommandMatch(t *testing.T) {
	v, w, x := newCommand("c", false).match(
		&patternList{newArgument("", "c")}, nil)
	y := patternList{newCommand("c", true)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newCommand("c", false).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newCommand("c", false).match(
		&patternList{
			newOption("-x", "", 0, false),
			newOption("-a", "", 0, false),
			newArgument("", "c")}, nil)
	y = patternList{newOption("-x", "", 0, false),
		newOption("-a", "", 0, false)}
	z := patternList{newCommand("c", true)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newEither(
		newCommand("add", false),
		newCommand("rm", false)).match(
		&patternList{newArgument("", "rm")}, nil)
	y = patternList{newCommand("rm", true)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}
}

func TestOptionalMatch(t *testing.T) {
	v, w, x := newOptional(newOption("-a", "", 0, false)).match(
		&patternList{newOption("-a", "", 0, false)}, nil)
	y := patternList{newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newOptional(newOption("-a", "", 0, false)).match(
		&patternList{}, nil)
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newOptional(newOption("-a", "", 0, false)).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newOptional(newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(
		&patternList{newOption("-a", "", 0, false)}, nil)
	y = patternList{newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newOptional(newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(
		&patternList{newOption("-b", "", 0, false)}, nil)
	y = patternList{newOption("-b", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newOptional(newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newOptional(newArgument("N", nil)).match(
		&patternList{newArgument("", 9)}, nil)
	y = patternList{newArgument("N", 9)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newOptional(newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(
		&patternList{newOption("-b", "", 0, false),
			newOption("-x", "", 0, false),
			newOption("-a", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	z := patternList{newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
}

func TestRequiredMatch(t *testing.T) {
	v, w, x := newRequired(newOption("-a", "", 0, false)).match(
		&patternList{newOption("-a", "", 0, false)}, nil)
	y := patternList{newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newRequired(newOption("-a", "", 0, false)).match(&patternList{}, nil)
	if v != false ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}

	v, w, x = newRequired(newOption("-a", "", 0, false)).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}
	v, w, x = newRequired(newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(
		&patternList{newOption("-a", "", 0, false)}, nil)
	y = patternList{newOption("-a", "", 0, false)}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, patternList{}) != true {
		t.Fail()
	}
}

func TestEitherMatch(t *testing.T) {
	v, w, x := newEither(
		newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(
		&patternList{newOption("-a", "", 0, false)}, nil)
	y := patternList{newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newEither(
		newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(&patternList{
		newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)}, nil)
	y = patternList{newOption("-b", "", 0, false)}
	z := patternList{newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newEither(
		newOption("-a", "", 0, false),
		newOption("-b", "", 0, false)).match(&patternList{
		newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	z = patternList{}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newEither(
		newOption("-a", "", 0, false),
		newOption("-b", "", 0, false),
		newOption("-c", "", 0, false)).match(&patternList{
		newOption("-x", "", 0, false),
		newOption("-b", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	z = patternList{newOption("-b", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
	v, w, x = newEither(
		newArgument("M", nil),
		newRequired(newArgument("N", nil),
			newArgument("M", nil))).match(&patternList{
		newArgument("", 1),
		newArgument("", 2)}, nil)
	y = patternList{}
	z = patternList{newArgument("N", 1), newArgument("M", 2)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
}

func TestOneOrMoreMatch(t *testing.T) {
	v, w, x := newOneOrMore(newArgument("N", nil)).match(
		&patternList{newArgument("", 9)}, nil)
	y := patternList{newArgument("N", 9)}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newArgument("N", nil)).match(
		&patternList{}, nil)
	y = patternList{}
	z := patternList{}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newArgument("N", nil)).match(
		&patternList{newOption("-x", "", 0, false)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	z = patternList{}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newArgument("N", nil)).match(
		&patternList{newArgument("", 9), newArgument("", 8)}, nil)
	y = patternList{}
	z = patternList{newArgument("N", 9), newArgument("N", 8)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newArgument("N", nil)).match(&patternList{
		newArgument("", 9),
		newOption("-x", "", 0, false),
		newArgument("", 8)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	z = patternList{newArgument("N", 9), newArgument("N", 8)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newOption("-a", "", 0, false)).match(&patternList{
		newOption("-a", "", 0, false),
		newArgument("", 8),
		newOption("-a", "", 0, false)}, nil)
	y = patternList{newArgument("", 8)}
	z = patternList{newOption("-a", "", 0, false), newOption("-a", "", 0, false)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newOption("-a", "", 0, false)).match(&patternList{
		newArgument("", 8),
		newOption("-x", "", 0, false)}, nil)
	y = patternList{newArgument("", 8), newOption("-x", "", 0, false)}
	z = patternList{}
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newRequired(newOption("-a", "", 0, false),
		newArgument("N", nil))).match(&patternList{
		newOption("-a", "", 0, false),
		newArgument("", 1),
		newOption("-x", "", 0, false),
		newOption("-a", "", 0, false),
		newArgument("", 2)}, nil)
	y = patternList{newOption("-x", "", 0, false)}
	z = patternList{newOption("-a", "", 0, false),
		newArgument("N", 1),
		newOption("-a", "", 0, false),
		newArgument("N", 2)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	v, w, x = newOneOrMore(newOptional(newArgument("N", nil))).match(
		&patternList{newArgument("", 9)}, nil)
	y = patternList{}
	z = patternList{newArgument("N", 9)}
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
}

func TestListArgumentMatch(t *testing.T) {
	p := newRequired(
		newArgument("N", nil),
		newArgument("N", nil))
	p.fix()
	v, w, x := p.match(&patternList{newArgument("", "1"),
		newArgument("", "2")}, nil)
	y := patternList{newArgument("N", []string{"1", "2"})}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	p = newOneOrMore(newArgument("N", nil))
	p.fix()
	v, w, x = p.match(&patternList{newArgument("", "1"),
		newArgument("", "2"), newArgument("", "3")}, nil)
	y = patternList{newArgument("N", []string{"1", "2", "3"})}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	p = newRequired(newArgument("N", nil),
		newOneOrMore(newArgument("N", nil)))
	p.fix()
	v, w, x = p.match(&patternList{
		newArgument("", "1"),
		newArgument("", "2"),
		newArgument("", "3")}, nil)
	y = patternList{newArgument("N", []string{"1", "2", "3"})}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	p = newRequired(newArgument("N", nil),
		newRequired(newArgument("N", nil)))
	p.fix()
	v, w, x = p.match(&patternList{
		newArgument("", "1"),
		newArgument("", "2")}, nil)
	y = patternList{newArgument("N", []string{"1", "2"})}
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}
}

func TestBasicPatternMatching(t *testing.T) {
	// ( -a N [ -x Z ] )
	p := newRequired(
		newOption("-a", "", 0, false),
		newArgument("N", nil),
		newOptional(
			newOption("-x", "", 0, false),
			newArgument("Z", nil)))

	// -a N
	q := patternList{newOption("-a", "", 0, false), newArgument("", 9)}
	y := patternList{newOption("-a", "", 0, false), newArgument("N", 9)}
	v, w, x := p.match(&q, nil)
	if v != true ||
		reflect.DeepEqual(*w, patternList{}) != true ||
		reflect.DeepEqual(*x, y) != true {
		t.Fail()
	}

	// -a -x N Z
	q = patternList{newOption("-a", "", 0, false),
		newOption("-x", "", 0, false),
		newArgument("", 9), newArgument("", 5)}
	y = patternList{}
	z := patternList{newOption("-a", "", 0, false), newArgument("N", 9),
		newOption("-x", "", 0, false), newArgument("Z", 5)}
	v, w, x = p.match(&q, nil)
	if v != true ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}

	// -x N Z  # BZZ!
	q = patternList{newOption("-x", "", 0, false),
		newArgument("", 9), newArgument("", 5)}
	y = patternList{newOption("-x", "", 0, false),
		newArgument("", 9), newArgument("", 5)}
	z = patternList{}
	v, w, x = p.match(&q, nil)
	if v != false ||
		reflect.DeepEqual(*w, y) != true ||
		reflect.DeepEqual(*x, z) != true {
		t.Fail()
	}
}

func TestPatternEither(t *testing.T) {
	p := newOption("-a", "", 0, false).transform()
	q := newEither(newRequired(
		newOption("-a", "", 0, false)))
	if p.eq(q) != true {
		t.Fail()
	}

	p = newArgument("A", nil).transform()
	q = newEither(newRequired(
		newArgument("A", nil)))
	if p.eq(q) != true {
		t.Fail()
	}

	p = newRequired(
		newEither(
			newOption("-a", "", 0, false),
			newOption("-b", "", 0, false)),
		newOption("-c", "", 0, false)).transform()
	q = newEither(
		newRequired(
			newOption("-a", "", 0, false),
			newOption("-c", "", 0, false)),
		newRequired(
			newOption("-b", "", 0, false),
			newOption("-c", "", 0, false)))
	if p.eq(q) != true {
		t.Fail()
	}

	p = newOptional(newOption("-a", "", 0, false),
		newEither(newOption("-b", "", 0, false),
			newOption("-c", "", 0, false))).transform()
	q = newEither(
		newRequired(
			newOption("-b", "", 0, false), newOption("-a", "", 0, false)),
		newRequired(
			newOption("-c", "", 0, false), newOption("-a", "", 0, false)))
	if p.eq(q) != true {
		t.Fail()
	}

	p = newEither(newOption("-x", "", 0, false),
		newEither(newOption("-y", "", 0, false),
			newOption("-z", "", 0, false))).transform()
	q = newEither(
		newRequired(newOption("-x", "", 0, false)),
		newRequired(newOption("-y", "", 0, false)),
		newRequired(newOption("-z", "", 0, false)))
	if p.eq(q) != true {
		t.Fail()
	}

	p = newOneOrMore(newArgument("N", nil),
		newArgument("M", nil)).transform()
	q = newEither(
		newRequired(newArgument("N", nil), newArgument("M", nil),
			newArgument("N", nil), newArgument("M", nil)))
	if p.eq(q) != true {
		t.Fail()
	}
}

func TestPatternFixRepeatingArguments(t *testing.T) {
	p := newOption("-a", "", 0, false)
	p.fixRepeatingArguments()
	if p.eq(newOption("-a", "", 0, false)) != true {
		t.Fail()
	}

	p = newArgument("N", nil)
	p.fixRepeatingArguments()
	if p.eq(newArgument("N", nil)) != true {
		t.Fail()
	}

	p = newRequired(
		newArgument("N", nil),
		newArgument("N", nil))
	q := newRequired(
		newArgument("N", []string{}),
		newArgument("N", []string{}))
	p.fixRepeatingArguments()
	if p.eq(q) != true {
		t.Fail()
	}

	p = newEither(
		newArgument("N", nil),
		newOneOrMore(newArgument("N", nil)))
	q = newEither(
		newArgument("N", []string{}),
		newOneOrMore(newArgument("N", []string{})))
	p.fix()
	if p.eq(q) != true {
		t.Fail()
	}
}

func TestSet(t *testing.T) {
	p := newArgument("N", nil)
	q := newArgument("N", nil)
	if reflect.DeepEqual(p, q) != true {
		t.Fail()
	}
	pl := patternList{newArgument("N", nil), newArgument("N", nil)}
	ql := patternList{newArgument("N", nil)}
	if reflect.DeepEqual(pl.unique(), ql.unique()) != true {
		t.Fail()
	}
}

func TestPatternFixIdentities1(t *testing.T) {
	p := newRequired(
		newArgument("N", nil),
		newArgument("N", nil))
	if len(p.children) < 2 {
		t.FailNow()
	}
	if p.children[0].eq(p.children[1]) != true {
		t.Fail()
	}
	if p.children[0] == p.children[1] {
		t.Fail()
	}
	p.fixIdentities(nil)
	if p.children[0] != p.children[1] {
		t.Fail()
	}
}

func TestPatternFixIdentities2(t *testing.T) {
	p := newRequired(
		newOptional(
			newArgument("X", nil),
			newArgument("N", nil)),
		newArgument("N", nil))
	if len(p.children) < 2 {
		t.FailNow()
	}
	if len(p.children[0].children) < 2 {
		t.FailNow()
	}
	if p.children[0].children[1].eq(p.children[1]) != true {
		t.Fail()
	}
	if p.children[0].children[1] == p.children[1] {
		t.Fail()
	}
	p.fixIdentities(nil)
	if p.children[0].children[1] != p.children[1] {
		t.Fail()
	}
}

func TestLongOptionsErrorHandling(t *testing.T) {
	_, err := Parse("Usage: prog", []string{"--non-existent"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(fmt.Sprintf("(%s) %s", reflect.TypeOf(err), err))
	}
	_, err = Parse("Usage: prog [--version --verbose]\nOptions: --version\n --verbose",
		[]string{"--ver"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
	_, err = Parse("Usage: prog --long\nOptions: --long ARG", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
	_, err = Parse("Usage: prog --long ARG\nOptions: --long ARG",
		[]string{"--long"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(fmt.Sprintf("(%s) %s", reflect.TypeOf(err), err))
	}
	_, err = Parse("Usage: prog --long=ARG\nOptions: --long", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
	_, err = Parse("Usage: prog --long\nOptions: --long",
		[]string{}, true, "--long=ARG", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
}

func TestShortOptionsErrorHandling(t *testing.T) {
	_, err := Parse("Usage: prog -x\nOptions: -x  this\n -x  that", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(fmt.Sprintf("(%s) %s", reflect.TypeOf(err), err))
	}
	_, err = Parse("Usage: prog", []string{"-x"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
	_, err = Parse("Usage: prog -o\nOptions: -o ARG", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
	_, err = Parse("Usage: prog -o ARG\nOptions: -o ARG", []string{"-o"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
}

func TestMatchingParen(t *testing.T) {
	_, err := Parse("Usage: prog [a [b]", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
	_, err = Parse("Usage: prog [a [b] ] c )", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
}

func TestAllowDoubleDash(t *testing.T) {
	if v, err := Parse("usage: prog [-o] [--] <arg>\noptions: -o", []string{"--", "-o"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-o": false, "<arg>": "-o", "--": true}) != true {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [-o] [--] <arg>\noptions: -o", []string{"-o", "1"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-o": true, "<arg>": "1", "--": false}) != true {
		t.Error(err)
	}
	_, err := Parse("usage: prog [-o] <arg>\noptions:-o", []string{"-o"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok { //"--" is not allowed; FIXME?
		t.Error(err)
	}
}

func TestDocopt(t *testing.T) {
	doc := `Usage: prog [-v] A

                Options: -v  Be verbose.`
	if v, err := Parse(doc, []string{"arg"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": false, "A": "arg"}) != true {
		t.Error(err)
	}
	if v, err := Parse(doc, []string{"-v", "arg"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": true, "A": "arg"}) != true {
		t.Error(err)
	}

	doc = `Usage: prog [-vqr] [FILE]
              prog INPUT OUTPUT
              prog --help

    Options:
      -v  print status messages
      -q  report only file names
      -r  show all occurrences of the same error
      --help

    `
	if v, err := Parse(doc, []string{"-v", "file.py"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": true, "-q": false, "-r": false, "--help": false, "FILE": "file.py", "INPUT": nil, "OUTPUT": nil}) != true {
		t.Error(err)
	}
	if v, err := Parse(doc, []string{"-v"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": true, "-q": false, "-r": false, "--help": false, "FILE": nil, "INPUT": nil, "OUTPUT": nil}) != true {
		t.Error(err)
	}

	_, err := Parse(doc, []string{"-v", "input.py", "output.py"}, true, "", false, false) // does not match
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
	_, err = Parse(doc, []string{"--fake"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
	_, output, err := parseOutput(doc, []string{"--hel"}, true, "", false)
	if err != nil || len(output) == 0 {
		t.Error(err)
	}
}

func TestLanguageErrors(t *testing.T) {
	_, err := Parse("no usage with colon here", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
	_, err = Parse("usage: here \n\n and again usage: here", []string{}, true, "", false, false)
	if _, ok := err.(*LanguageError); !ok {
		t.Error(err)
	}
}

func TestIssue40(t *testing.T) {
	_, output, err := parseOutput("usage: prog --help-commands | --help", []string{"--help"}, true, "", false)
	if err != nil || len(output) == 0 {
		t.Error(err)
	}
	if v, err := Parse("usage: prog --aabb | --aa", []string{"--aa"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--aabb": false, "--aa": true}) != true {
		t.Error(err)
	}
}

func TestIssue34UnicodeStrings(t *testing.T) {
	// TODO: see if applicable
}

func TestCountMultipleFlags(t *testing.T) {
	if v, err := Parse("usage: prog [-v]", []string{"-v"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": true}) != true {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [-vv]", []string{}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": 0}) != true {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [-vv]", []string{"-v"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": 1}) != true {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [-vv]", []string{"-vv"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": 2}) != true {
		t.Error(err)
	}
	_, err := Parse("usage: prog [-vv]", []string{"-vvv"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [-v | -vv | -vvv]", []string{"-vvv"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": 3}) != true {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [-v...]", []string{"-vvvvvv"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-v": 6}) != true {
		t.Error(err)
	}
	if v, err := Parse("usage: prog [--ver --ver]", []string{"--ver", "--ver"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--ver": 2}) != true {
		t.Error(err)
	}
}

func TestAnyOptionsParameter(t *testing.T) {
	_, err := Parse("usage: prog [options]",
		[]string{"-foo", "--bar", "--spam=eggs"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Fail()
	}

	_, err = Parse("usage: prog [options]",
		[]string{"--foo", "--bar", "--bar"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Fail()
	}
	_, err = Parse("usage: prog [options]",
		[]string{"--bar", "--bar", "--bar", "-ffff"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Fail()
	}
	_, err = Parse("usage: prog [options]",
		[]string{"--long=arg", "--long=another"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Fail()
	}
}

func TestDefaultValueForPositionalArguments(t *testing.T) {
	doc := "Usage: prog [--data=<data>...]\nOptions:\n\t-d --data=<arg>    Input data [default: x]"
	if v, err := Parse(doc, []string{}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--data": []string{"x"}}) != true {
		t.Error(err)
	}

	doc = "Usage: prog [--data=<data>...]\nOptions:\n\t-d --data=<arg>    Input data [default: x y]"
	if v, err := Parse(doc, []string{}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--data": []string{"x", "y"}}) != true {
		t.Error(err)
	}

	doc = "Usage: prog [--data=<data>...]\nOptions:\n\t-d --data=<arg>    Input data [default: x y]"
	if v, err := Parse(doc, []string{"--data=this"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--data": []string{"this"}}) != true {
		t.Error(err)
	}
}

func TestIssue59(t *testing.T) {
	if v, err := Parse("usage: prog --long=<a>", []string{"--long="}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--long": ""}) != true {
		t.Error(err)
	}

	if v, err := Parse("usage: prog -l <a>\noptions: -l <a>", []string{"-l", ""}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"-l": ""}) != true {
		t.Error(err)
	}
}

func TestOptionsFirst(t *testing.T) {
	if v, err := Parse("usage: prog [--opt] [<args>...]", []string{"--opt", "this", "that"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--opt": true, "<args>": []string{"this", "that"}}) != true {
		t.Error(err)
	}

	if v, err := Parse("usage: prog [--opt] [<args>...]", []string{"this", "that", "--opt"}, true, "", false, false); reflect.DeepEqual(v, map[string]interface{}{"--opt": true, "<args>": []string{"this", "that"}}) != true {
		t.Error(err)
	}

	if v, err := Parse("usage: prog [--opt] [<args>...]", []string{"this", "that", "--opt"}, true, "", true, false); reflect.DeepEqual(v, map[string]interface{}{"--opt": false, "<args>": []string{"this", "that", "--opt"}}) != true {
		t.Error(err)
	}
}

func TestIssue68OptionsShortcutDoesNotIncludeOptionsInUsagePattern(t *testing.T) {
	args, err := Parse("usage: prog [-ab] [options]\noptions: -x\n -y", []string{"-ax"}, true, "", false, false)

	if args["-a"] != true {
		t.Error(err)
	}
	if args["-b"] != false {
		t.Error(err)
	}
	if args["-x"] != true {
		t.Error(err)
	}
	if args["-y"] != false {
		t.Error(err)
	}
}

func TestIssue65EvaluateArgvWhenCalledNotWhenImported(t *testing.T) {
	os.Args = strings.Fields("prog -a")
	v, err := Parse("usage: prog [-ab]", nil, true, "", false, false)
	w := map[string]interface{}{"-a": true, "-b": false}
	if reflect.DeepEqual(v, w) != true {
		t.Error(err)
	}

	os.Args = strings.Fields("prog -b")
	v, err = Parse("usage: prog [-ab]", nil, true, "", false, false)
	w = map[string]interface{}{"-a": false, "-b": true}
	if reflect.DeepEqual(v, w) != true {
		t.Error(err)
	}
}

func TestIssue71DoubleDashIsNotAValidOptionArgument(t *testing.T) {
	_, err := Parse("usage: prog [--log=LEVEL] [--] <args>...",
		[]string{"--log", "--", "1", "2"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Fail()
	}

	_, err = Parse(`usage: prog [-l LEVEL] [--] <args>...
                  options: -l LEVEL`, []string{"-l", "--", "1", "2"}, true, "", false, false)
	if _, ok := err.(*UserError); !ok {
		t.Fail()
	}
}

func TestParseSection(t *testing.T) {
	v := parseSection("usage:", "foo bar fizz buzz")
	w := []string{}
	if reflect.DeepEqual(v, w) != true {
		t.Fail()
	}

	v = parseSection("usage:", "usage: prog")
	w = []string{"usage: prog"}
	if reflect.DeepEqual(v, w) != true {
		t.Fail()
	}

	v = parseSection("usage:", "usage: -x\n -y")
	w = []string{"usage: -x\n -y"}
	if reflect.DeepEqual(v, w) != true {
		t.Fail()
	}

	usage := `usage: this

usage:hai
usage: this that

usage: foo
       bar

PROGRAM USAGE:
 foo
 bar
usage:
` + "\t" + `too
` + "\t" + `tar
Usage: eggs spam
BAZZ
usage: pit stop`

	v = parseSection("usage:", usage)
	w = []string{"usage: this",
		"usage:hai",
		"usage: this that",
		"usage: foo\n       bar",
		"PROGRAM USAGE:\n foo\n bar",
		"usage:\n\ttoo\n\ttar",
		"Usage: eggs spam",
		"usage: pit stop",
	}
	if reflect.DeepEqual(v, w) != true {
		t.Fail()
	}
}

func TestIssue126DefaultsNotParsedCorrectlyWhenTabs(t *testing.T) {
	section := "Options:\n\t--foo=<arg>  [default: bar]"
	v := patternList{newOption("", "--foo", 1, "bar")}
	if reflect.DeepEqual(parseDefaults(section), v) != true {
		t.Fail()
	}
}

// conf file based test cases
func TestFileTestcases(t *testing.T) {
	filenames := []string{"testcases.docopt", "test_golang.docopt"}
	for _, filename := range filenames {
		raw, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}

		tests, err := parseTest(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range tests {
			result, err := Parse(c.doc, c.argv, true, "", false, false)
			if _, ok := err.(*UserError); c.userError && !ok {
				// expected a user-error
				t.Error("testcase:", c.id, "result:", result)
			} else if _, ok := err.(*UserError); !c.userError && ok {
				// unexpected user-error
				t.Error("testcase:", c.id, "error:", err, "result:", result)
			} else if reflect.DeepEqual(c.expect, result) != true {
				t.Error("testcase:", c.id, "result:", result, "expect:", c.expect)
			}
		}
	}
}

type testcase struct {
	id        int
	doc       string
	prog      string
	argv      []string
	expect    map[string]interface{}
	userError bool
}

func parseTest(raw []byte) ([]testcase, error) {
	var res []testcase
	commentPattern := regexp.MustCompile("#.*")
	raw = commentPattern.ReplaceAll(raw, []byte(""))
	raw = bytes.TrimSpace(raw)
	if bytes.HasPrefix(raw, []byte(`"""`)) {
		raw = raw[3:]
	}

	id := 0
	for _, fixture := range bytes.Split(raw, []byte(`r"""`)) {
		doc, _, body := stringPartition(string(fixture), `"""`)
		for _, cas := range strings.Split(body, "$")[1:] {
			argvString, _, expectString := stringPartition(strings.TrimSpace(cas), "\n")
			prog, _, argvString := stringPartition(strings.TrimSpace(argvString), " ")
			argv := []string{}
			if len(argvString) > 0 {
				argv = strings.Fields(argvString)
			}
			var expectUntyped interface{}
			err := json.Unmarshal([]byte(expectString), &expectUntyped)
			if err != nil {
				return nil, err
			}
			switch expect := expectUntyped.(type) {
			case string: // user-error
				res = append(res, testcase{id, doc, prog, argv, nil, true})
			case map[string]interface{}:
				// convert []interface{} values to []string
				// convert float64 values to int
				for k, vUntyped := range expect {
					switch v := vUntyped.(type) {
					case []interface{}:
						itemList := make([]string, len(v))
						for i, itemUntyped := range v {
							if item, ok := itemUntyped.(string); ok {
								itemList[i] = item
							}
						}
						expect[k] = itemList
					case float64:
						expect[k] = int(v)
					}
				}
				res = append(res, testcase{id, doc, prog, argv, expect, false})
			default:
				return nil, fmt.Errorf("unhandled json data type")
			}
			id++
		}
	}
	return res, nil
}

// parseOutput wraps the Parse() function to also return stdout
func parseOutput(doc string, argv []string, help bool, version string,
	optionsFirst bool) (map[string]interface{}, string, error) {
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	args, err := Parse(doc, argv, help, version, optionsFirst, false)

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	w.Close()
	os.Stdout = stdout
	output := <-outChan

	return args, output, err
}

var debugEnabled = false

func debugOn(l ...interface{}) {
	debugEnabled = true
	debug(l...)
}
func debugOff(l ...interface{}) {
	debug(l...)
	debugEnabled = false
}

func debug(l ...interface{}) {
	if debugEnabled {
		fmt.Println(l...)
	}
}
