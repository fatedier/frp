# curl http://uov.ecoheze.com/frp/install_frps.sh | sudo bash
frps_down_url="http://uov.ecoheze.com/frp"


arch=`uname -p`
echo "设备架构类型: $arch"
if [ "$arch" == "x86_64" ]; then
    frps_file_name="frps_linux_amd64.tar.gz"
    frps_path_name="frps_linux_amd64"
elif [ "$arch" == "aarch64" ]; then
    frps_file_name="frps_linux_arm64.tar.gz"
    frps_path_name="frps_linux_arm64"
else
    echo "未知架构设备: $arch"
    exit
fi

frps_file_url=$frps_down_url/$frps_file_name

echo "frps使用文件的地址:" $frps_file_url

wget $frps_file_url -O /tmp/$frps_file_name
if [ -e "/tmp/$frps_file_name" ]; then
    echo "frps下载完成"
else
    echo "frps下载失败"
    exit 1
fi

tar -zxf "/tmp/$frps_file_name" -C /tmp/
if [ -e "/tmp/$frps_path_name/frps" ]; then
    echo "frps解压完成"
else
    echo "frps解压失败"
    exit 1
fi
chown root:root /tmp/$frps_path_name/frps
chmod +x /tmp/$frps_path_name/frps
mv -f /tmp/$frps_path_name/frps /bin/

if [ -e "/bin/frps.toml" ]; then
    echo "/bin/frps.toml文件存在"
else
    echo "/bin/frps.toml文件不存在"
    mv -f /tmp/$frps_path_name/frps.toml /bin/
fi

rm -rf /tmp/$frps_file_name
rm -rf /tmp/$frps_path_name

# 查看是ubuntu还是centos
os=`cat /etc/os-release | grep ^ID= | awk -F= '{print $2}'`
echo "设备系统类型: $os"
if [ "$os" == "ubuntu" ]; then
    service_file="/lib/systemd/system/frps.service"
elif [ "$os" == "\"centos\"" ]; then
    service_file="/usr/lib/systemd/system/frps.service"
else
    echo "未知系统类型: $os"
    exit
fi

echo "[Unit]
Description = frps server
After = network.target syslog.target
Wants = network.target

[Service]
Type=simple
ExecStart = /bin/frps -c /bin/frps.toml

[Install]
WantedBy=multi-user.target
" > $service_file


systemctl daemon-reload

systemctl enable frps
systemctl start frps
systemctl status frps


echo "ok"