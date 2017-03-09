#!/bin/bash

./bin/echo_server &
./bin/http_server &
./../bin/frps -c ./conf/auto_test_frps.ini &
sleep 1
./../bin/frpc -c ./conf/auto_test_frpc.ini &

# wait until proxies are connected
for((i=1; i<15; i++))
do
    sleep 1
    str=`ss -ant|grep 10700|grep LISTEN`
    if [ -z "${str}" ]; then
        echo "wait"
        continue
    fi

    str=`ss -ant|grep 10710|grep LISTEN`
    if [ -z "${str}" ]; then
        echo "wait"
        continue
    fi

    str=`ss -ant|grep 10711|grep LISTEN`
    if [ -z "${str}" ]; then
        echo "wait"
        continue
    fi

    break
done

sleep 1
