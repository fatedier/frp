#!/bin/bash

pid=`ps aux|grep './../bin/frps -c ./conf/auto_test_frps.ini'|grep -v grep|awk {'print $2'}`
if [ -n "${pid}" ]; then
    kill ${pid}
fi

pid=`ps aux|grep './../bin/frpc -c ./conf/auto_test_frpc.ini'|grep -v grep|awk {'print $2'}`
if [ -n "${pid}" ]; then
    kill ${pid}
fi

pid=`ps aux|grep './../bin/frpc -c ./conf/auto_test_frpc_visitor.ini'|grep -v grep|awk {'print $2'}`
if [ -n "${pid}" ]; then
    kill ${pid}
fi

rm -f ./frps.log
rm -f ./frpc.log
rm -f ./frpc_visitor.log
