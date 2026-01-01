// Copyright 2021 The frp Authors
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

package sub

import (
	"errors"
	"fmt"
	"github.com/fatedier/frp/pkg/policy/security"
	"github.com/fatedier/frp/pkg/util/log/events"
	"github.com/fatedier/frp/pkg/util/system"
	"github.com/fatedier/frp/pkg/util/version"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/pkg/config"
	"github.com/fatedier/frp/pkg/config/v1/validation"
)

var (
	verifyInstallation = false
	restricted         = false
)

const fmtServiceDesc = "Frp is a fast reverse proxy that allows you to expose a local server located behind a NAT or firewall to the Internet. This service is %s."

func init() {
	installCmd.PersistentFlags().BoolVarP(&verifyInstallation, "verify", "", false, "verify config(s) before installation")
	installCmd.PersistentFlags().BoolVarP(&restricted, "restricted", "", false, "run service in restricted context")

	installCmd.PersistentFlags().StringSliceVarP(&allowUnsafe, "allow-unsafe", "", []string{},
		fmt.Sprintf("allowed unsafe features, one or more of: %s", strings.Join(security.ClientUnsafeFeatures, ", ")))

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install this executable as a Windows service or update the existing service (run as privileged user)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		execPath, err := os.Executable()
		if err != nil {
			fmt.Println("frpc: the executable no longer exists")
			os.Exit(1)
		}
		stat, err := os.Stat(execPath)
		if err != nil || stat.IsDir() {
			fmt.Println("frpc: the executable is no longer valid")
			os.Exit(1)
		}

		// Ignore other params if "--config-dir" specified
		if cfgDir != "" {
			if verifyInstallation {
				var hasValidCfg = false
				err := filepath.WalkDir(cfgDir, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return os.ErrNotExist
					}
					if d.IsDir() {
						return nil
					}
					cfgFile1 := cfgDir + "\\" + d.Name()
					if verifyCfg(cfgFile1) == nil {
						fmt.Printf("frpc: the configuration file %s syntax is ok\n", cfgFile1)
						hasValidCfg = true
					}
					return nil
				})
				if !hasValidCfg {
					err = errors.New("no valid configuration file found")
				}
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
			err := installService(execPath, "--config-dir", cfgDir)
			if err != nil {
				os.Exit(1)
			}
			return nil
		}

		// Ignore other params if "-c" / "--config" specified
		if cfgFile != "" {
			if verifyInstallation {
				err := verifyCfg(cfgFile)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				fmt.Printf("frpc: the configuration file %s syntax is ok\n", cfgFile)
			}
			err := installService(execPath, "--config", cfgFile)
			if err != nil {
				os.Exit(1)
			}
			return nil
		}

		err = installService(execPath, args...)
		if err != nil {
			os.Exit(1)
		}

		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the Windows service installed for this executable (run as privileged user)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		err := uninstallService()
		if err != nil {
			os.Exit(1)
		}

		return nil
	},
}

func verifyCfg(f string) error {
	cliCfg, proxyCfgs, visitorCfgs, _, err := config.LoadClientConfig(f, strictConfigMode)
	if err != nil {
		return err
	}
	unsafeFeatures := security.NewUnsafeFeatures(allowUnsafe)
	warning, err := validation.ValidateAllClientConfig(cliCfg, proxyCfgs, visitorCfgs, unsafeFeatures)
	if warning != nil {
		fmt.Printf("WARNING: %v\n", warning)
	}
	if err != nil {
		return err
	}
	return nil
}

func installService(exec string, args ...string) error {
	scm, err := mgr.Connect()
	if err != nil {
		fmt.Println("frpc: Failed connect to SCM (Permission Denied)")
		return err
	}
	defer func() {
		_ = scm.Disconnect()
	}()

	// Check and modify existing service
	if modifyService(scm, exec, args...) == nil {
		return nil
	}
	// Create new service
	var objName string
	var sidType uint32 = windows.SERVICE_SID_TYPE_UNRESTRICTED
	if restricted {
		objName = "NT AUTHORITY\\LocalService"
		sidType = windows.SERVICE_SID_TYPE_RESTRICTED
	}
	_, err = scm.CreateService("frpc", exec, mgr.Config{
		ErrorControl:     mgr.ErrorNormal,
		TagId:            0,
		Dependencies:     []string{"Tcpip"},
		ServiceStartName: objName,
		DisplayName:      system.ServiceName,
		Description:      fmt.Sprintf(fmtServiceDesc, system.ServiceName),
		SidType:          sidType,
		DelayedAutoStart: false,
	}, args...)
	if err != nil {
		return err
	}
	_ = events.CreateEventSource(system.ServiceName)
	fmt.Println("Service successfully installed.")
	return nil
}

func modifyService(scm *mgr.Mgr, exec string, args ...string) error {
	service, err := scm.OpenService("frpc")
	if err != nil {
		return err
	}
	defer func(service *mgr.Service) {
		_ = service.Close()
	}(service)
	serviceConfig, err := service.Config()
	if err != nil {
		return err
	}
	s := syscall.EscapeArg(exec)
	for _, v := range args {
		s += " " + syscall.EscapeArg(v)
	}
	serviceConfig.BinaryPathName = s
	var objName string
	var sidType uint32 = windows.SERVICE_SID_TYPE_UNRESTRICTED
	if restricted {
		objName = "NT AUTHORITY\\LocalService"
		sidType = windows.SERVICE_SID_TYPE_RESTRICTED
	}
	serviceConfig.ServiceStartName = objName
	serviceConfig.SidType = sidType
	err = service.UpdateConfig(serviceConfig)
	if err != nil {
		return err
	}
	return nil
}

func uninstallService() error {
	scm, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer func() {
		_ = scm.Disconnect()
	}()

	service, err := scm.OpenService("frpc")
	if err != nil {
		return err
	}
	defer func(service *mgr.Service) {
		_ = service.Close()
	}(service)
	err = service.Delete()
	if err != nil {
		return err
	}
	_ = events.DeleteEventSource(system.ServiceName)
	fmt.Println("Service successfully removed.")
	return nil
}
