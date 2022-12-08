#!/bin/bash
# set -x
declare -i cache
declare -i clean
declare -i loop
host=`ifconfig enp1s0 | grep -w "inet" | awk '/inet/ {print $2}'`
echo ${host}
loop=1
RUN(){
    echo "[INFO]Clean all docker cache before test ..."
    docker system prune -af

    echo "[INFO]Warm up " $1 " ..."
    time docker build -q --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .

    for ((i=1; i<=${loop}; i++))do      
        while true;do
            echo "[INFO]Clean the test image before " $1 "-" ${i} " default test ..."
            re=`docker rmi -f test`  
            startTraffic1=`cat /proc/net/dev | grep ens | awk '{print $2}'`
            re=`time docker build -q -t test -f Dockerfile . `
            endTraffic1=`cat /proc/net/dev | grep ens | awk '{print $2}'`
            echo "[INFO]Test " $1 "-" ${i} ", default = " $((${endTraffic1} - ${startTraffic1})) 
            if [[ $re =~ "sha256" ]];then
                echo $re
                break
            else
                echo "Error occurs in default test, possibly dur to unstable network, the image build will be restart ..."
            fi
        done

        while true;do
            echo "[INFO]Clean the test image before " $1 "-" ${i}  " cached test ..."
            re=`docker rmi -f test`
            startTraffic2=`cat /proc/net/dev | grep ens | awk '{print $2}'`
            re=`time docker build -q --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .`
            endTraffic2=`cat /proc/net/dev | grep ens | awk '{print $2}'`
            echo "[INFO]Test " $1 "-" ${i} ", cached = " $((${endTraffic2} - ${startTraffic2})) 
            if [[ $re =~ "sha256" ]];then
                echo $re
                break
            else
                echo "Error occurs in cached test, possibly dur to unstable network, the image build will be restart ..."
            fi
        done
    done
}

RUNALL(){
    for file in *;do
        if [ -d ${file} ];then   #dir
                cd ${file} # enter dir
                if [ -f "Dockerfile" ];then
                    RUN ${file} ${loop}
                fi
                cd ..
        fi
    done
}

# set -x
if [ $# -eq 0 ]; then
    RUNALL
elif [ $# -eq 1 ]; then
    if [ -d $1 ];then   #dir
            cd $1 # enter dir
            if [ -f "Dockerfile" ];then
                RUN $1 ${loop}
            fi
            cd ..
    else
        loop=$1
        RUNALL
    fi
else
    loop=$2
    if [ -d $1 ];then   #dir
        cd $1 # enter dir
        if [ -f "Dockerfile" ];then
            RUN $1 ${loop}
        fi
        cd ..
    fi
fi
