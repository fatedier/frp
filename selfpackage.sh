#!/bin/sh
set -e

# compile for version
make
if [ $? -ne 0 ]; then
    echo "make error"
    exit 1
fi

frp_version=`./bin/frps --version`
echo "build version: $frp_version"

# cross_compiles
make -f ./Makefile.cross-compiles

rm -rf ./release/packages
mkdir -p ./release/packages

os_all='linux windows android'
arch_all='amd64 arm arm64'
extra_all='_ hf'

cd ./release

for os in $os_all; do
    for arch in $arch_all; do
        for extra in $extra_all; do
            suffix="${os}_${arch}"
            if [ "x${extra}" != x"_" ]; then
                suffix="${os}_${arch}_${extra}"
            fi
            frps_dir_name="frps_${suffix}"
            frps_path="./packages/${frps_dir_name}"
            frpc_dir_name="frpc_${suffix}"
            frpc_path="./packages/${frpc_dir_name}"

            if [ "x${os}" = x"windows" ]; then
                if [ ! -f "./frpc_${os}_${arch}.exe" ]; then
                    continue
                fi
                if [ ! -f "./frps_${os}_${arch}.exe" ]; then
                    continue
                fi
                mkdir ${frps_path}
                mkdir ${frpc_path}
                mv ./frpc_${os}_${arch}.exe ${frpc_path}/frpc.exe
                mv ./frps_${os}_${arch}.exe ${frps_path}/frps.exe
            else
                if [ ! -f "./frpc_${suffix}" ]; then
                    continue
                fi
                if [ ! -f "./frps_${suffix}" ]; then
                    continue
                fi
                mkdir ${frps_path}
                mkdir ${frpc_path}
                mv ./frpc_${suffix} ${frpc_path}/frpc
                mv ./frps_${suffix} ${frps_path}/frps
            fi

            cp -f ../conf/frpc.toml ${frpc_path}
            cp -f ../conf/frpc1.toml ${frpc_path}
            cp -f ../conf/frps.toml ${frps_path}

            # packages
            cd ./packages
            if [ "x${os}" = x"windows" ]; then
                zip -rq ${frps_dir_name}.zip ${frps_dir_name}
                zip -rq ${frpc_dir_name}.zip ${frpc_dir_name}
            else
                tar -zcf ${frps_dir_name}.tar.gz ${frps_dir_name}
                tar -zcf ${frpc_dir_name}.tar.gz ${frpc_dir_name}
            fi
            cd ..
            rm -rf ${frpc_path}
            rm -rf ${frps_path}
        done
    done
done

cd -
