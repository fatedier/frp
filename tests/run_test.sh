#!/bin/bash

./../bin/frps -c ./conf/auto_test_frps.ini &
sleep 1
./../bin/frpc -c ./conf/auto_test_frpc.ini &

# wait until proxies are connected
sleep 2
