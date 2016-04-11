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

rm -rf ./packages
mkdir ./packages

os_all='linux windows'
arch_all='386 amd64'

for os in $os_all; do
    for arch in $arch_all; do
        frp_dir_name="frp_${frp_version}_${os}_${arch}"
        frp_path="./packages/frp_${frp_version}_${os}_${arch}"
        mkdir ${frp_path}
        if [ "x${os}" = x"windows" ]; then
            mv ./frpc_${os}_${arch}.exe ${frp_path}/frpc.exe
            mv ./frps_${os}_${arch}.exe ${frp_path}/frps.exe
        else
            mv ./frpc_${os}_${arch} ${frp_path}/frpc
            mv ./frps_${os}_${arch} ${frp_path}/frps
        fi  
        cp ./LICENSE ${frp_path}
        cp ./conf/* ${frp_path}

        # packages
        cd ./packages
        if [ "x${os}" = x"windows" ]; then
            zip -rq ${frp_dir_name}.zip ${frp_dir_name}
        else
            tar -zcf ${frp_dir_name}.tar.gz ${frp_dir_name}
        fi  
        cd ..
        rm -rf ${frp_path}
    done
done
