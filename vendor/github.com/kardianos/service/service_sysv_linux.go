// Copyright 2015 Daniel Theophanes.
// Use of this source code is governed by a zlib-style
// license that can be found in the LICENSE file.

package service

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/template"
	"time"
)

type sysv struct {
	i        Interface
	platform string
	*Config
}

func newSystemVService(i Interface, platform string, c *Config) (Service, error) {
	s := &sysv{
		i:        i,
		platform: platform,
		Config:   c,
	}

	return s, nil
}

func (s *sysv) String() string {
	if len(s.DisplayName) > 0 {
		return s.DisplayName
	}
	return s.Name
}

func (s *sysv) Platform() string {
	return s.platform
}

var errNoUserServiceSystemV = errors.New("User services are not supported on SystemV.")

func (s *sysv) configPath() (cp string, err error) {
	if s.Option.bool(optionUserService, optionUserServiceDefault) {
		err = errNoUserServiceSystemV
		return
	}
	cp = "/etc/init.d/" + s.Config.Name
	return
}

func (s *sysv) template() *template.Template {
	customScript := s.Option.string(optionSysvScript, "")

	if customScript != "" {
		return template.Must(template.New("").Funcs(tf).Parse(customScript))
	} else {
		return template.Must(template.New("").Funcs(tf).Parse(sysvScript))
	}
}

func (s *sysv) Install() error {
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
		Path string
	}{
		s.Config,
		path,
	}

	err = s.template().Execute(f, to)
	if err != nil {
		return err
	}

	if err = os.Chmod(confPath, 0755); err != nil {
		return err
	}
	for _, i := range [...]string{"2", "3", "4", "5"} {
		if err = os.Symlink(confPath, "/etc/rc"+i+".d/S50"+s.Name); err != nil {
			continue
		}
	}
	for _, i := range [...]string{"0", "1", "6"} {
		if err = os.Symlink(confPath, "/etc/rc"+i+".d/K02"+s.Name); err != nil {
			continue
		}
	}

	return nil
}

func (s *sysv) Uninstall() error {
	cp, err := s.configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(cp); err != nil {
		return err
	}
	return nil
}

func (s *sysv) Logger(errs chan<- error) (Logger, error) {
	if system.Interactive() {
		return ConsoleLogger, nil
	}
	return s.SystemLogger(errs)
}
func (s *sysv) SystemLogger(errs chan<- error) (Logger, error) {
	return newSysLogger(s.Name, errs)
}

func (s *sysv) Run() (err error) {
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

func (s *sysv) Status() (Status, error) {
	_, out, err := runWithOutput("service", s.Name, "status")
	if err != nil {
		return StatusUnknown, err
	}

	switch {
	case strings.HasPrefix(out, "Running"):
		return StatusRunning, nil
	case strings.HasPrefix(out, "Stopped"):
		return StatusStopped, nil
	default:
		return StatusUnknown, ErrNotInstalled
	}
}

func (s *sysv) Start() error {
	return run("service", s.Name, "start")
}

func (s *sysv) Stop() error {
	return run("service", s.Name, "stop")
}

func (s *sysv) Restart() error {
	err := s.Stop()
	if err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)
	return s.Start()
}

const sysvScript = `#!/bin/sh
# For RedHat and cousins:
# chkconfig: - 99 01
# description: {{.Description}}
# processname: {{.Path}}

### BEGIN INIT INFO
# Provides:          {{.Path}}
# Required-Start:
# Required-Stop:
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: {{.DisplayName}}
# Description:       {{.Description}}
### END INIT INFO

cmd="{{.Path}}{{range .Arguments}} {{.|cmd}}{{end}}"

name=$(basename $(readlink -f $0))
pid_file="/var/run/$name.pid"
stdout_log="/var/log/$name.log"
stderr_log="/var/log/$name.err"

[ -e /etc/sysconfig/$name ] && . /etc/sysconfig/$name

get_pid() {
    cat "$pid_file"
}

is_running() {
    [ -f "$pid_file" ] && ps $(get_pid) > /dev/null 2>&1
}

case "$1" in
    start)
        if is_running; then
            echo "Already started"
        else
            echo "Starting $name"
            {{if .WorkingDirectory}}cd '{{.WorkingDirectory}}'{{end}}
            $cmd >> "$stdout_log" 2>> "$stderr_log" &
            echo $! > "$pid_file"
            if ! is_running; then
                echo "Unable to start, see $stdout_log and $stderr_log"
                exit 1
            fi
        fi
    ;;
    stop)
        if is_running; then
            echo -n "Stopping $name.."
            kill $(get_pid)
            for i in $(seq 1 10)
            do
                if ! is_running; then
                    break
                fi
                echo -n "."
                sleep 1
            done
            echo
            if is_running; then
                echo "Not stopped; may still be shutting down or shutdown may have failed"
                exit 1
            else
                echo "Stopped"
                if [ -f "$pid_file" ]; then
                    rm "$pid_file"
                fi
            fi
        else
            echo "Not running"
        fi
    ;;
    restart)
        $0 stop
        if is_running; then
            echo "Unable to stop, will not attempt to start"
            exit 1
        fi
        $0 start
    ;;
    status)
        if is_running; then
            echo "Running"
        else
            echo "Stopped"
            exit 1
        fi
    ;;
    *)
    echo "Usage: $0 {start|stop|restart|status}"
    exit 1
    ;;
esac
exit 0
`
