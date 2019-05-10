// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

package service

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

func isSystemd() bool {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	return false
}

type systemd struct {
	i        Interface
	platform string
	*Config
}

func newSystemdService(i Interface, platform string, c *Config) (Service, error) {
	s := &systemd{
		i:        i,
		platform: platform,
		Config:   c,
	}

	return s, nil
}

func (s *systemd) String() string {
	if len(s.DisplayName) > 0 {
		return s.DisplayName
	}
	return s.Name
}

func (s *systemd) Platform() string {
	return s.platform
}

// Systemd services should be supported, but are not currently.
var errNoUserServiceSystemd = errors.New("User services are not supported on systemd.")

func (s *systemd) configPath() (cp string, err error) {
	if s.Option.bool(optionUserService, optionUserServiceDefault) {
		err = errNoUserServiceSystemd
		return
	}
	cp = "/etc/systemd/system/" + s.Config.Name + ".service"
	return
}

func (s *systemd) getSystemdVersion() int64 {
	_, out, err := runWithOutput("systemctl", "--version")
	if err != nil {
		return -1
	}

	re := regexp.MustCompile(`systemd ([0-9]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) != 2 {
		return -1
	}

	v, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return -1
	}

	return v
}

func (s *systemd) hasOutputFileSupport() bool {
	defaultValue := true
	version := s.getSystemdVersion()
	if version == -1 {
		return defaultValue
	}

	if version < 236 {
		return false
	}

	return defaultValue
}

func (s *systemd) template() *template.Template {
	customScript := s.Option.string(optionSystemdScript, "")

	if customScript != "" {
		return template.Must(template.New("").Funcs(tf).Parse(customScript))
	} else {
		return template.Must(template.New("").Funcs(tf).Parse(systemdScript))
	}
}

func (s *systemd) Install() error {
	confPath, err := s.configPath()
	if err != nil {
		return err
	}
	_, err = os.Stat(confPath)
	if err == nil {
		return fmt.Errorf("Init already exists: %s", confPath)
	}

	f, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer f.Close()

	path, err := s.execPath()
	if err != nil {
		return err
	}

	var to = &struct {
		*Config
		Path                 string
		HasOutputFileSupport bool
		ReloadSignal         string
		PIDFile              string
		LogOutput            bool
	}{
		s.Config,
		path,
		s.hasOutputFileSupport(),
		s.Option.string(optionReloadSignal, ""),
		s.Option.string(optionPIDFile, ""),
		s.Option.bool(optionLogOutput, optionLogOutputDefault),
	}

	err = s.template().Execute(f, to)
	if err != nil {
		return err
	}

	err = run("systemctl", "enable", s.Name+".service")
	if err != nil {
		return err
	}
	return run("systemctl", "daemon-reload")
}

func (s *systemd) Uninstall() error {
	err := run("systemctl", "disable", s.Name+".service")
	if err != nil {
		return err
	}
	cp, err := s.configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(cp); err != nil {
		return err
	}
	return nil
}

func (s *systemd) Logger(errs chan<- error) (Logger, error) {
	if system.Interactive() {
		return ConsoleLogger, nil
	}
	return s.SystemLogger(errs)
}
func (s *systemd) SystemLogger(errs chan<- error) (Logger, error) {
	return newSysLogger(s.Name, errs)
}

func (s *systemd) Run() (err error) {
	err = s.i.Start(s)
	if err != nil {
		return err
	}

	s.Option.funcSingle(optionRunWait, func() {
		var sigChan = make(chan os.Signal, 3)
		signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)
		<-sigChan
	})()

	return s.i.Stop(s)
}

func (s *systemd) Status() (Status, error) {
	exitCode, out, err := runWithOutput("systemctl", "is-active", s.Name)
	if exitCode == 0 && err != nil {
		return StatusUnknown, err
	}

	switch {
	case strings.HasPrefix(out, "active"):
		return StatusRunning, nil
	case strings.HasPrefix(out, "inactive"):
		return StatusStopped, nil
	case strings.HasPrefix(out, "failed"):
		return StatusUnknown, errors.New("service in failed state")
	default:
		return StatusUnknown, ErrNotInstalled
	}
}

func (s *systemd) Start() error {
	return run("systemctl", "start", s.Name+".service")
}

func (s *systemd) Stop() error {
	return run("systemctl", "stop", s.Name+".service")
}

func (s *systemd) Restart() error {
	return run("systemctl", "restart", s.Name+".service")
}

const systemdScript = `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}

[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
{{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
{{if and .LogOutput .HasOutputFileSupport -}}
StandardOutput=file:/var/log/{{.Name}}.out
StandardError=file:/var/log/{{.Name}}.err
{{- end}}
Restart=always
RestartSec=120
EnvironmentFile=-/etc/sysconfig/{{.Name}}

[Install]
WantedBy=multi-user.target
`
