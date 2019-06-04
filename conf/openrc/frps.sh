#!/sbin/openrc-run
# Copyright 1999-2013 Gentoo Foundation
# Distributed under the terms of the GNU General Public License v2

pidfile="/run/frps.pid"
configfile="/etc/frp/frps.ini"
command="/usr/bin/frps"
command_opts=" -c $configfile"

depend() {
    need net
    need localmount
}

start() {
    ebegin "Starting frps"
    start-stop-daemon --start --background \
    --exec $command \
    -- $command_opts \
    --make-pidfile --pidfile $pidfile
    eend $?
}

stop() {
    ebegin "Stopping frps"
    start-stop-daemon --stop \
    --exec $command \
    --pidfile $pidfile
    eend $?
}

reload() {
    ebegin "Reloading frps"
    start-stop-daemon --exec $command \
    --pidfile $pidfile \
    -s 1
    eend $?
}