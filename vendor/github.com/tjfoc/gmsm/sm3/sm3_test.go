/*
Copyright Suzhou Tongji Fintech Research Institute 2017 All Rights Reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sm3

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func byteToString(b []byte) string {
	ret := ""
	for i := 0; i < len(b); i++ {
		ret += fmt.Sprintf("%02x", b[i])
	}
	return ret
}
func TestSm3(t *testing.T) {
	msg := []byte("test")
	err := ioutil.WriteFile("ifile", msg, os.FileMode(0644)) // 生成测试文件
	if err != nil {
		log.Fatal(err)
	}
	msg, err = ioutil.ReadFile("ifile")
	if err != nil {
		log.Fatal(err)
	}
	hw := New()
	hw.Write(msg)
	hash := hw.Sum(nil)
	fmt.Println(hash)
	fmt.Printf("%s\n", byteToString(hash))
	hash1 := Sm3Sum(msg)
	fmt.Println(hash1)
	fmt.Printf("%s\n", byteToString(hash1))

}

func BenchmarkSm3(t *testing.B) {
	t.ReportAllocs()
	msg := []byte("test")
	hw := New()
	for i := 0; i < t.N; i++ {

		hw.Sum(nil)
		Sm3Sum(msg)
	}
}
