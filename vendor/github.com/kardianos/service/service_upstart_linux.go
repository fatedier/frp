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
	"strings"
	"syscall"
	"text/template"
	"time"
)

func isUpstart() bool {
	if _, err := os.Stat("/sbin/upstart-udev-bridge"); err == nil {
		return true
	}
	if _, err := os.Stat("/sbin/initctl"); err == nil {
		if _, out, err := runWithOutput("/sbin/initctl", "--version"); err == nil {
			if strings.Contains(out, "initctl (upstart") {
				return true
			}
		}
	}
	return false
}

type upstart struct {
	i        Interface
	platform string
	*Config
}

func newUpstartService(i Interface, platform string, c *Config) (Service, error) {
	s := &upstart{
		i:        i,
		platform: platform,
		Config:   c,
	}

	return s, nil
}

func (s *upstart) String() string {
	if len(s.DisplayName) > 0 {
		return s.DisplayName
	}
	return s.Name
}

func (s *upstart) Platform() string {
	return s.platform
}

// Upstart has some support for user services in graphical sessions.
// Due to the mix of actual support for user services over versions, just don't bother.
// Upstart will be replaced by systemd in most cases anyway.
var errNoUserServiceUpstart = errors.New("User services are not supported on Upstart.")

func (s *upstart) configPath() (cp string, err error) {
	if s.Option.bool(optionUserService, optionUserServiceDefault) {
		err = errNoUserServiceUpstart
		return
	}
	cp = "/etc/init/" + s.Config.Name + ".conf"
	return
}

func (s *upstart) hasKillStanza() bool {
	defaultValue := true
	version := s.getUpstartVersion()
	if version == nil {
		return defaultValue
	}

	maxVersion := []int{0, 6, 5}
	if matches, err := versionAtMost(version, maxVersion); err != nil || matches {
		return false
	}

	return defaultValue
}

func (s *upstart) hasSetUIDStanza() bool {
	defaultValue := true
	version := s.getUpstartVersion()
	if version == nil {
		return defaultValue
	}

	maxVersion := []int{1, 4, 0}
	if matches, err := versionAtMost(version, maxVersion); err != nil || matches {
		return false
	}

	return defaultValue
}

func (s *upstart) getUpstartVersion() []int {
	_, out, err := runWithOutput("/sbin/initctl", "--version")
	if err != nil {
		return nil
	}

	re := regexp.MustCompile(`initctl \(upstart (\d+.\d+.\d+)\)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) != 2 {
		return nil
	}

	return parseVersion(matches[1])
}

func (s *upstart) template() *template.Template {
	customScript := s.Option.string(optionUpstartScript, "")

	if customScript != "" {
		return template.Must(template.New("").Funcs(tf).Parse(customScript))
	} else {
		return template.Must(template.New("").Funcs(tf).Parse(upstartScript))
	}
}

func (s *upstart) Install() error {
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
		Path            string
		HasKillStanza   bool
		HasSetUIDStanza bool
		LogOutput       bool
	}{
		s.Config,
		path,
		s.hasKillStanza(),
		s.hasSetUIDStanza(),
		s.Option.bool(optionLogOutput, optionLogOutputDefault),
	}

	return s.template().Execute(f, to)
}

func (s *upstart) Uninstall() error {
	cp, err := s.configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(cp); err != nil {
		return err
	}
	return nil
}

func (s *upstart) Logger(errs chan<- error) (Logger, error) {
	if system.Interactive() {
		return ConsoleLogger, nil
	}
	return s.SystemLogger(errs)
}
func (s *upstart) SystemLogger(errs chan<- error) (Logger, error) {
	return newSysLogger(s.Name, errs)
}

func (s *upstart) Run() (err error) {
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

func (s *upstart) Status() (Status, error) {
	exitCode, out, err := runWithOutput("initctl", "status", s.Name)
	if exitCode == 0 && err != nil {
		return StatusUnknown, err
	}

	switch {
	case strings.HasPrefix(out, fmt.Sprintf("%s start/running", s.Name)):
		return StatusRunning, nil
	case strings.HasPrefix(out, fmt.Sprintf("%s stop/waiting", s.Name)):
		return StatusStopped, nil
	default:
		return StatusUnknown, ErrNotInstalled
	}
}

func (s *upstart) Start() error {
	return run("initctl", "start", s.Name)
}

func (s *upstart) Stop() error {
	return run("initctl", "stop", s.Name)
}

func (s *upstart) Restart() error {
	err := s.Stop()
	if err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return s.Start()
}

// The upstart script should stop with an INT or the Go runtime will terminate
// the program before the Stop handler can run.
const upstartScript = `# {{.Description}}

{{if .DisplayName}}description    "{{.DisplayName}}"{{end}}

{{if .HasKillStanza}}kill signal INT{{end}}
{{if .ChRoot}}chroot {{.ChRoot}}{{end}}
{{if .WorkingDirectory}}chdir {{.WorkingDirectory}}{{end}}
start on filesystem or runlevel [2345]
stop on runlevel [!2345]

{{if and .UserName .HasSetUIDStanza}}setuid {{.UserName}}{{end}}

respawn
respawn limit 10 5
umask 022

console none

pre-start script
    test -x {{.Path}} || { stop; exit 0; }
end script

# Start
script
	{{if .LogOutput}}
	stdout_log="/var/log/{{.Name}}.out"
	stderr_log="/var/log/{{.Name}}.err"
	{{end}}
	
	if [ -f "/etc/sysconfig/{{.Name}}" ]; then
		set -a
		source /etc/sysconfig/{{.Name}}
		set +a
	fi

	exec {{if and .UserName (not .HasSetUIDStanza)}}sudo -E -u {{.UserName}} {{end}}{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}{{if .LogOutput}} >> $stdout_log 2>> $stderr_log{{end}}
end script
`
