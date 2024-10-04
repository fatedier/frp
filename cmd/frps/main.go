// Copyright 2018 fatedier, fatedier@gmail.com
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
	"os"

	_ "github.com/fatedier/frp/assets/frps"
	_ "github.com/fatedier/frp/pkg/metrics"
	"github.com/fatedier/frp/pkg/util/system"
	"github.com/fatedier/frp/pkg/util/version"
)

func main() {
	fmt.Println(`
 __    __                      ________                   
|  \  |  \                    |        \                  
| $$  | $$  ______   __    __ | $$$$$$$$______    ______  
| $$__| $$ |      \ |  \  |  \| $$__   /      \  /      \ 
| $$    $$  \$$$$$$\| $$  | $$| $$  \ |  $$$$$$\|  $$$$$$\
| $$$$$$$$ /      $$| $$  | $$| $$$$$ | $$   \$$| $$  | $$
| $$  | $$|  $$$$$$$| $$__/ $$| $$    | $$      | $$__/ $$
| $$  | $$ \$$    $$ \$$    $$| $$    | $$      | $$    $$
 \$$   \$$  \$$$$$$$ _\$$$$$$$ \$$     \$$      | $$$$$$$ 
                    |  \__| $$                  | $$      
                     \$$    $$                  | $$      
                      \$$$$$$                    \$$               ——— HayFrp公益项目运营&开发组
					  
	`)
	fmt.Println("欢迎使用HayFrp！")
	fmt.Println("HayFrp程序发行版本：" + version.Full())
	system.EnableCompatibilityMode()

	// 检查是否有frps.ini文件
	if _, err := os.Stat("frps.ini"); err == nil {
		// 如果有，将-c frps.ini添加到os.Args中
		os.Args = append(os.Args, "-c", "frps.ini")
	}

	Execute()
}
