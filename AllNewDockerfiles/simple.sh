#!/bin/bash
# set -x
declare -i cache
declare -i clean
declare -i loop
host=`ifconfig enp1s0 | grep -w "inet" | awk '/inet/ {print $2}'`
echo ${host}
loop=1
RUN(){
    echo "[INFO]Clean all docker images before test ..."
    re=`docker system prune -af`

    echo "[INFO]Try " $1 " ..."
    time docker build --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .
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
# ./test.sh
# ./test.sh 5
# ./test.sh zookeeper_latest
# ./test.sh zookeeper_latest 5
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
