#!/bin/bash
# set -x
declare -i cache
declare -i clean
declare -i loop
host=` ifconfig ens33 | awk '/inet/ {print $2}' | cut -f2 -d ":"`
echo ${host}
loop=1
RUN(){
    echo "[INFO]Clean all docker cache before test ..."
    docker system prune -af

    echo "[INFO]Warm up " $1 " ..."
    time docker build --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .

    for ((i=1; i<=${loop}; i++));do
        echo "[INFO]Clean the test image before " ${i} " default test ..."
        re=`docker rmi -f test`

        startTraffic1=`cat /proc/net/dev | grep ens | awk '{print $2}'`
        time docker build -t test -f Dockerfile . 
        endTraffic1=`cat /proc/net/dev | grep ens | awk '{print $2}'`
        echo "[INFO]Test " $1 "-" ${i} ", default = " $((${endTraffic1} - ${startTraffic1})) 

        echo "[INFO]Clean the test image before " ${i} " cached test ..."
        re=`docker rmi -f test`

        startTraffic2=`cat /proc/net/dev | grep ens | awk '{print $2}'`
        time docker build -q --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .
        endTraffic2=`cat /proc/net/dev | grep ens | awk '{print $2}'`
        echo "[INFO]Test " $1 "-" ${i} ", cached = " $((${endTraffic2} - ${startTraffic2})) 
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
