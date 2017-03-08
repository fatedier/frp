package net

import (
	"fmt"
	"testing"
)

func TestA(t *testing.T) {
	l, err := ListenUDP("0.0.0.0", 9000)
	if err != nil {
		fmt.Println(err)
	}
	for {
		c, _ := l.Accept()
		go func() {
			for {
				buf := make([]byte, 1450)
				n, err := c.Read(buf)
				if err != nil {
					fmt.Println(buf[:n])
				}

				c.Write(buf[:n])
			}
		}()
	}
}
