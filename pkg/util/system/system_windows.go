// Copyright 2024 The frp Authors
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

package system

import (
	"fmt"
	"golang.org/x/sys/windows/svc"
	"os"
)

var ServiceName = ""

// EnableCompatibilityMode enables compatibility mode for different system.
// For example, on Android, the inability to obtain the correct time zone will result in incorrect log time output.
func EnableCompatibilityMode() {
}

// Run wraps Execute function for different system.
// For example, on Windows, it runs as a Windows service if necessary.
func Run(name string, f func()) {
	ServiceName = name
	// Check if we are running as a Windows service.
	inService, err := svc.IsWindowsService()
	if err != nil {
		os.Exit(1)
	} else if inService {
		// Start as a service.
		err := svc.Run(name, &frpService{
			f: f,
		})
		if err != nil {
			os.Exit(1)
		}
	} else {
		// Run as a usual program.
		f()
	}
}

type frpService struct {
	f func()
}

func (f frpService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	s <- svc.Status{State: svc.StartPending}

	defer func() {
		s <- svc.Status{State: svc.StopPending}
		fmt.Println("Stopping service...")
	}()

	// Main function.
	if len(args) > 1 {
		// Replace all parameters if specified in Services MMC
		os.Args = append(os.Args[:1], args[1:]...)
	}
	go f.f()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown | svc.AcceptParamChange}
	fmt.Println("Service started.")

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				return
			case svc.Interrogate:
				s <- c.CurrentStatus
			case svc.ParamChange:
				fmt.Println("Reloading configurations...")
				// TODO: Trigger reloading
			default:
				fmt.Printf("ERROR: Unexpected services control request #%d\n", c)
			}
		}
	}
}
