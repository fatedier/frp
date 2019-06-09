#!/sbin/openrc-run
# Copyright 1999-2013 Gentoo Foundation
# Distributed under the terms of the GNU General Public License v2

pidfile="/run/frpc.pid"
configfile="/etc/frp/frpc.ini"
command="/usr/bin/frpc"
command_opts=" -c $configfile"

depend() {
    need net
    need localmount
}

start() {
    ebegin "Starting frpc"
    start-stop-daemon --start --background \
    --make-pidfile --pidfile $pidfile \
    --exec $command \
    -- $command_opts
    eend $?
}

stop() {
    ebegin "Stopping frpc"
    start-stop-daemon --stop \
    --exec $command \
    --pidfile $pidfile
    eend $?
}

reload() {
    ebegin "Reloading frpc"
    start-stop-daemon --exec $command \
    --pidfile $pidfile \
    -s 1
    eend $?
}