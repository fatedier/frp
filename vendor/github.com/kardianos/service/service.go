// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

// Package service provides a simple way to create a system service.
// Currently supports Windows, Linux/(systemd | Upstart | SysV), and OSX/Launchd.
//
// Windows controls services by setting up callbacks that is non-trivial. This
// is very different then other systems. This package provides the same API
// despite the substantial differences.
// It also can be used to detect how a program is called, from an interactive
// terminal or from a service manager.
//
// Examples in the example/ folder.
//
//	package main
//
//	import (
//		"log"
//
//		"github.com/kardianos/service"
//	)
//
//	var logger service.Logger
//
//	type program struct{}
//
//	func (p *program) Start(s service.Service) error {
//		// Start should not block. Do the actual work async.
//		go p.run()
//		return nil
//	}
//	func (p *program) run() {
//		// Do work here
//	}
//	func (p *program) Stop(s service.Service) error {
//		// Stop should not block. Return with a few seconds.
//		return nil
//	}
//
//	func main() {
//		svcConfig := &service.Config{
//			Name:        "GoServiceTest",
//			DisplayName: "Go Service Test",
//			Description: "This is a test Go service.",
//		}
//
//		prg := &program{}
//		s, err := service.New(prg, svcConfig)
//		if err != nil {
//			log.Fatal(err)
//		}
//		logger, err = s.Logger(nil)
//		if err != nil {
//			log.Fatal(err)
//		}
//		err = s.Run()
//		if err != nil {
//			logger.Error(err)
//		}
//	}
package service // import "github.com/kardianos/service"

import (
	"errors"
	"fmt"
)

const (
	optionKeepAlive            = "KeepAlive"
	optionKeepAliveDefault     = true
	optionRunAtLoad            = "RunAtLoad"
	optionRunAtLoadDefault     = false
	optionUserService          = "UserService"
	optionUserServiceDefault   = false
	optionSessionCreate        = "SessionCreate"
	optionSessionCreateDefault = false
	optionLogOutput            = "LogOutput"
	optionLogOutputDefault     = false

	optionRunWait      = "RunWait"
	optionReloadSignal = "ReloadSignal"
	optionPIDFile      = "PIDFile"

	optionSystemdScript = "SystemdScript"
	optionSysvScript    = "SysvScript"
	optionUpstartScript = "UpstartScript"
	optionLaunchdConfig = "LaunchdConfig"
)

// Status represents service status as an byte value
type Status byte

// Status of service represented as an byte
const (
	StatusUnknown Status = iota // Status is unable to be determined due to an error or it was not installed.
	StatusRunning
	StatusStopped
)

// Config provides the setup for a Service. The Name field is required.
type Config struct {
	Name        string   // Required name of the service. No spaces suggested.
	DisplayName string   // Display name, spaces allowed.
	Description string   // Long description of service.
	UserName    string   // Run as username.
	Arguments   []string // Run with arguments.

	// Optional field to specify the executable for service.
	// If empty the current executable is used.
	Executable string

	// Array of service dependencies.
	// Not yet implemented on Linux or OS X.
	Dependencies []string

	// The following fields are not supported on Windows.
	WorkingDirectory string // Initial working directory.
	ChRoot           string

	// System specific options.
	//  * OS X
	//    - LaunchdConfig string ()      - Use custom launchd config
	//    - KeepAlive     bool   (true)
	//    - RunAtLoad     bool   (false)
	//    - UserService   bool   (false) - Install as a current user service.
	//    - SessionCreate bool   (false) - Create a full user session.
	//  * POSIX
	//    - SystemdScript string ()                 - Use custom systemd script
	//    - UpstartScript string ()                 - Use custom upstart script
	//    - SysvScript    string ()                 - Use custom sysv script
	//    - RunWait       func() (wait for SIGNAL)  - Do not install signal but wait for this function to return.
	//    - ReloadSignal  string () [USR1, ...]     - Signal to send on reaload.
	//    - PIDFile       string () [/run/prog.pid] - Location of the PID file.
	//    - LogOutput     bool   (false)            - Redirect StdErr & StdOut to files.
	Option KeyValue
}

var (
	system         System
	systemRegistry []System
)

var (
	// ErrNameFieldRequired is returned when Config.Name is empty.
	ErrNameFieldRequired = errors.New("Config.Name field is required.")
	// ErrNoServiceSystemDetected is returned when no system was detected.
	ErrNoServiceSystemDetected = errors.New("No service system detected.")
	// ErrNotInstalled is returned when the service is not installed
	ErrNotInstalled = errors.New("the service is not installed")
)

// New creates a new service based on a service interface and configuration.
func New(i Interface, c *Config) (Service, error) {
	if len(c.Name) == 0 {
		return nil, ErrNameFieldRequired
	}
	if system == nil {
		return nil, ErrNoServiceSystemDetected
	}
	return system.New(i, c)
}

// KeyValue provides a list of platform specific options. See platform docs for
// more details.
type KeyValue map[string]interface{}

// bool returns the value of the given name, assuming the value is a boolean.
// If the value isn't found or is not of the type, the defaultValue is returned.
func (kv KeyValue) bool(name string, defaultValue bool) bool {
	if v, found := kv[name]; found {
		if castValue, is := v.(bool); is {
			return castValue
		}
	}
	return defaultValue
}

// int returns the value of the given name, assuming the value is an int.
// If the value isn't found or is not of the type, the defaultValue is returned.
func (kv KeyValue) int(name string, defaultValue int) int {
	if v, found := kv[name]; found {
		if castValue, is := v.(int); is {
			return castValue
		}
	}
	return defaultValue
}

// string returns the value of the given name, assuming the value is a string.
// If the value isn't found or is not of the type, the defaultValue is returned.
func (kv KeyValue) string(name string, defaultValue string) string {
	if v, found := kv[name]; found {
		if castValue, is := v.(string); is {
			return castValue
		}
	}
	return defaultValue
}

// float64 returns the value of the given name, assuming the value is a float64.
// If the value isn't found or is not of the type, the defaultValue is returned.
func (kv KeyValue) float64(name string, defaultValue float64) float64 {
	if v, found := kv[name]; found {
		if castValue, is := v.(float64); is {
			return castValue
		}
	}
	return defaultValue
}

// funcSingle returns the value of the given name, assuming the value is a float64.
// If the value isn't found or is not of the type, the defaultValue is returned.
func (kv KeyValue) funcSingle(name string, defaultValue func()) func() {
	if v, found := kv[name]; found {
		if castValue, is := v.(func()); is {
			return castValue
		}
	}
	return defaultValue
}

// Platform returns a description of the system service.
func Platform() string {
	if system == nil {
		return ""
	}
	return system.String()
}

// Interactive returns false if running under the OS service manager
// and true otherwise.
func Interactive() bool {
	if system == nil {
		return true
	}
	return system.Interactive()
}

func newSystem() System {
	for _, choice := range systemRegistry {
		if choice.Detect() == false {
			continue
		}
		return choice
	}
	return nil
}

// ChooseSystem chooses a system from the given system services.
// SystemServices are considered in the order they are suggested.
// Calling this may change what Interactive and Platform return.
func ChooseSystem(a ...System) {
	systemRegistry = a
	system = newSystem()
}

// ChosenSystem returns the system that service will use.
func ChosenSystem() System {
	return system
}

// AvailableSystems returns the list of system services considered
// when choosing the system service.
func AvailableSystems() []System {
	return systemRegistry
}

// System represents the service manager that is available.
type System interface {
	// String returns a description of the system.
	String() string

	// Detect returns true if the system is available to use.
	Detect() bool

	// Interactive returns false if running under the system service manager
	// and true otherwise.
	Interactive() bool

	// New creates a new service for this system.
	New(i Interface, c *Config) (Service, error)
}

// Interface represents the service interface for a program. Start runs before
// the hosting process is granted control and Stop runs when control is returned.
//
//   1. OS service manager executes user program.
//   2. User program sees it is executed from a service manager (IsInteractive is false).
//   3. User program calls Service.Run() which blocks.
//   4. Interface.Start() is called and quickly returns.
//   5. User program runs.
//   6. OS service manager signals the user program to stop.
//   7. Interface.Stop() is called and quickly returns.
//      - For a successful exit, os.Exit should not be called in Interface.Stop().
//   8. Service.Run returns.
//   9. User program should quickly exit.
type Interface interface {
	// Start provides a place to initiate the service. The service doesn't not
	// signal a completed start until after this function returns, so the
	// Start function must not take more then a few seconds at most.
	Start(s Service) error

	// Stop provides a place to clean up program execution before it is terminated.
	// It should not take more then a few seconds to execute.
	// Stop should not call os.Exit directly in the function.
	Stop(s Service) error
}

// TODO: Add Configure to Service interface.

// Service represents a service that can be run or controlled.
type Service interface {
	// Run should be called shortly after the program entry point.
	// After Interface.Stop has finished running, Run will stop blocking.
	// After Run stops blocking, the program must exit shortly after.
	Run() error

	// Start signals to the OS service manager the given service should start.
	Start() error

	// Stop signals to the OS service manager the given service should stop.
	Stop() error

	// Restart signals to the OS service manager the given service should stop then start.
	Restart() error

	// Install setups up the given service in the OS service manager. This may require
	// greater rights. Will return an error if it is already installed.
	Install() error

	// Uninstall removes the given service from the OS service manager. This may require
	// greater rights. Will return an error if the service is not present.
	Uninstall() error

	// Opens and returns a system logger. If the user program is running
	// interactively rather then as a service, the returned logger will write to
	// os.Stderr. If errs is non-nil errors will be sent on errs as well as
	// returned from Logger's functions.
	Logger(errs chan<- error) (Logger, error)

	// SystemLogger opens and returns a system logger. If errs is non-nil errors
	// will be sent on errs as well as returned from Logger's functions.
	SystemLogger(errs chan<- error) (Logger, error)

	// String displays the name of the service. The display name if present,
	// otherwise the name.
	String() string

	// Platform displays the name of the system that manages the service.
	// In most cases this will be the same as service.Platform().
	Platform() string

	// Status returns the current service status.
	Status() (Status, error)
}

// ControlAction list valid string texts to use in Control.
var ControlAction = [5]string{"start", "stop", "restart", "install", "uninstall"}

// Control issues control functions to the service from a given action string.
func Control(s Service, action string) error {
	var err error
	switch action {
	case ControlAction[0]:
		err = s.Start()
	case ControlAction[1]:
		err = s.Stop()
	case ControlAction[2]:
		err = s.Restart()
	case ControlAction[3]:
		err = s.Install()
	case ControlAction[4]:
		err = s.Uninstall()
	default:
		err = fmt.Errorf("Unknown action %s", action)
	}
	if err != nil {
		return fmt.Errorf("Failed to %s %v: %v", action, s, err)
	}
	return nil
}

// Logger writes to the system log.
type Logger interface {
	Error(v ...interface{}) error
	Warning(v ...interface{}) error
	Info(v ...interface{}) error

	Errorf(format string, a ...interface{}) error
	Warningf(format string, a ...interface{}) error
	Infof(format string, a ...interface{}) error
}
