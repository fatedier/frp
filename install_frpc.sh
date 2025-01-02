# curl http://uov.ecoheze.com/frp/install_frpc.sh | sudo bash
frpc_down_url="http://uov.ecoheze.com/frp"


arch=`uname -p`
echo "设备架构类型: $arch"
if [ "$arch" == "x86_64" ]; then
    frpc_file_name="frpc_linux_amd64.tar.gz"
    frpc_path_name="frpc_linux_amd64"
elif [ "$arch" == "aarch64" ]; then
    frpc_file_name="frpc_linux_arm64.tar.gz"
    frpc_path_name="frpc_linux_arm64"
else
    echo "未知架构设备: $arch"
    exit
fi

frpc_file_url=$frpc_down_url/$frpc_file_name

echo "frpc使用文件的地址:" $frpc_file_url

wget $frpc_file_url -O /tmp/$frpc_file_name
if [ -e "/tmp/$frpc_file_name" ]; then
    echo "frpc下载完成"
else
    echo "frpc下载失败"
    exit 1
fi

tar -zxf "/tmp/$frpc_file_name" -C /tmp/
if [ -e "/tmp/$frpc_path_name/frpc" ]; then
    echo "frpc解压完成"
else
    echo "frpc解压失败"
    exit 1
fi
chown root:root /tmp/$frpc_path_name/frpc
chmod +x /tmp/$frpc_path_name/frpc
mv -f /tmp/$frpc_path_name/frpc /bin/

if [ -e "/bin/frpc.toml" ]; then
    echo "/bin/frpc.toml文件存在"
else
    echo "/bin/frpc.toml文件不存在"
    mv -f /tmp/$frpc_path_name/frpc.toml /bin/
fi

if [ -e "/bin/frpc1.toml" ]; then
    echo "/bin/frpc1.toml文件存在"
else
    echo "/bin/frpc1.toml文件不存在"
    mv -f /tmp/$frpc_path_name/frpc1.toml /bin/
fi

rm -rf /tmp/$frpc_file_name
rm -rf /tmp/$frpc_path_name

# 查看是ubuntu还是centos
os=`cat /etc/os-release | grep ^ID= | awk -F= '{print $2}'`
echo "设备系统类型: $os"
if [ "$os" == "ubuntu" ]; then
    service_file="/lib/systemd/system/frpc.service"
    service_file1="/lib/systemd/system/frpc1.service"
elif [ "$os" == "\"centos\"" ]; then
    service_file="/usr/lib/systemd/system/frpc.service"
    service_file1="/usr/lib/systemd/system/frpc1.service"
else
    echo "未知系统类型: $os"
    exit
fi

echo "[Unit]
Description = frpc server
After = network.target syslog.target
Wants = network.target

[Service]
Type=simple
WorkingDirectory=/bin
ExecStart = /bin/frpc

[Install]
WantedBy=multi-user.target
" > $service_file

echo "[Unit]
Description = frpc1 server
After = network.target syslog.target
Wants = network.target

[Service]
Type=simple
WorkingDirectory=/bin
ExecStart = /bin/frpc -t frp1

[Install]
WantedBy=multi-user.target
" > $service_file1

systemctl daemon-reload

systemctl enable frpc
systemctl stop frpc
systemctl start frpc
systemctl status frpc

systemctl enable frpc1
systemctl stop frpc1
systemctl start frpc1
systemctl status frpc1

echo "ok"