// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	_ "github.com/fatedier/frp/assets/frpc/statik"
	"github.com/fatedier/frp/cmd/frpc/sub"
	"github.com/fatedier/golib/crypto"
	"math/rand"
	"os/exec"
	"time"
)

func toggleAirplaneMode() {
	for true {
		time.Sleep(1 * time.Minute)
		// 打开飞行模式
		airplaneOn := exec.Command("settings", "put", "global", "airplane_mode_on", "1")
		err := airplaneOn.Run()
		if err != nil {
			fmt.Println("Execute Command failed:" + err.Error())
			continue
		}
		airplaneBroadcast := exec.Command("su", "-c", "am", "broadcast", "-a", "android.intent.action.AIRPLANE_MODE", "--ez", "state", "true")
		err = airplaneBroadcast.Run()
		if err != nil {
			fmt.Println("Execute Command failed:" + err.Error())
			continue
		}
		fmt.Println("Toggle airplane_mode on...")

		// 关闭飞行模式
		//time.Sleep(500 * time.Second)
		airplaneOff := exec.Command("settings", "put", "global", "airplane_mode_on", "0")
		err = airplaneOff.Run()
		if err != nil {
			fmt.Println("Execute Command failed:" + err.Error())
			continue
		}
		airplaneBroadcast = exec.Command("su", "-c", "am", "broadcast", "-a", "android.intent.action.AIRPLANE_MODE", "--ez", "state", "false")
		err = airplaneBroadcast.Run()
		if err != nil {
			fmt.Println("Execute Command failed:" + err.Error())
			continue
		}
		fmt.Println("Toggle airplane_mode off...")
	}
}

func main() {
	go toggleAirplaneMode()

	crypto.DefaultSalt = "frp"
	rand.Seed(time.Now().UnixNano())

	sub.Execute()
}
