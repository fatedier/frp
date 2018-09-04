package main

import (
	"flag"
	"fmt"
	"os"
)

var vects = flag.Uint64("vects", 20, "number of vects (data+parity)")
var data = flag.Uint64("data", 0, "number of data vects; keep it empty if you want to "+
	"get the max num of inverse matrix")

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println("  cntinverse [-flags]")
		fmt.Println("  Valid flags:")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if *vects > 256 {
		fmt.Println("Error: vects must <= 256")
		os.Exit(1)
	}
	if *data == 0 {
		n := getMAXCCombination(*vects)
		fmt.Println("max num of inverse matrix :", n)
		os.Exit(0)
	}
	n := getCCombination(*vects, *data)
	fmt.Println("num of inverse matrix:", n)
	os.Exit(0)
}

func getMAXCCombination(a uint64) uint64 {
	b := a / 2 // proved in mathtool/combination.jpg
	return getCCombination(a, b)
}

func getCCombination(a, b uint64) uint64 {
	top := make([]uint64, a-b)
	bottom := make([]uint64, a-b-1)
	for i := b + 1; i <= a; i++ {
		top[i-b-1] = i
	}
	var i uint64
	for i = 2; i <= a-b; i++ {
		bottom[i-2] = i
	}
	for j := 0; j <= 5; j++ {
		cleanEven(top, bottom)
		clean3(top, bottom)
		clean5(top, bottom)
	}
	cleanCoffeRound1(top, bottom)
	if maxBottomBigger5more1(bottom) {
		top = shuffTop(top)
		cleanCoffeRound1(top, bottom)
		cleanCoffeRound1(bottom, top)
		cleanCoffeRound1(top, bottom)
		cleanCoffeRound1(bottom, top)
		cleanCoffeRound1(top, bottom)
		cleanCoffeRound1(bottom, top)
	}
	var topV, bottomV uint64 = 1, 1
	for _, t := range top {
		topV = topV * t
	}
	for _, b := range bottom {
		bottomV = bottomV * b
	}
	return topV / bottomV
}

func cleanEven(top, bottom []uint64) {
	for i, b := range bottom {
		if even(b) {
			for j, t := range top {
				if even(t) {
					top[j] = t / 2
					bottom[i] = b / 2
					break
				}
			}
		}
	}
}

func even(a uint64) bool {
	return a&1 == 0
}

func clean3(top, bottom []uint64) {
	for i, b := range bottom {
		if mod3(b) {
			for j, t := range top {
				if mod3(t) {
					top[j] = t / 3
					bottom[i] = b / 3
					break
				}
			}
		}
	}
}

func mod3(a uint64) bool {
	c := a / 3
	if 3*c == a {
		return true
	}
	return false
}

func clean5(top, bottom []uint64) {
	for i, b := range bottom {
		if mod5(b) {
			for j, t := range top {
				if mod5(t) {
					top[j] = t / 5
					bottom[i] = b / 5
					break
				}
			}
		}
	}
}

func mod5(a uint64) bool {
	c := a / 5
	if 5*c == a {
		return true
	}
	return false
}

func maxBottomBigger5more1(bottom []uint64) bool {
	cnt := 0
	for _, b := range bottom {
		if b >= 5 {
			cnt++
		}
	}
	if cnt >= 2 {
		return true
	}
	return false
}

func cleanCoffeRound1(top, bottom []uint64) {
	for i, b := range bottom {
		for j, t := range top {
			if isCoffe(b, t) {
				top[j] = t / b
				bottom[i] = 1
				break
			}
		}
	}
}

func isCoffe(b, t uint64) bool {
	c := t / b
	if c*b == t {
		return true
	}
	return false
}

func shuffTop(top []uint64) []uint64 {
	var tmp uint64 = 1
	newLen := len(top) + 1
	for i, t := range top {
		if t <= 5 {
			tmp = tmp * t
			newLen--
			top[i] = 1
		}
	}
	topNew := make([]uint64, newLen)
	topNew[0] = tmp
	cnt := 1
	for _, t := range top {
		if t != 1 {
			topNew[cnt] = t
			cnt++
		}
	}
	return topNew
}
