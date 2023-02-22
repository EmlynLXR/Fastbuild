#!/bin/bash
# docker rmi test ; time docker build -q -t test -f Dockerfile . ; docker rmi test ; time docker build -q --build-arg http_proxy=http://202.114.6.221:8080 --build-arg HTTP_PROXY=http://202.114.6.221:8080 -t test -f Dockerfile .
# set -x
declare -i cache
declare -i clean
declare -i loop
host=`ifconfig enp1s0 | grep -w "inet" | awk '/inet/ {print $2}'`
echo ${host}
loop=1
lists=('nginx_latest' 'openjdk_oraclelinux8' 'redis_7.0.5' 'zookeeper_latest') 
# False
# 'debian_11.5' 'ubuntu_22.10'
# Not sure
# 'bitnami_kafka_3.3.1' 'bitnami_spark_latest'
# True
# 'flink_latest' 'storm_latest'
RUN(){
    # echo "[INFO]Clean all docker images before test ..."
    # re=`docker system prune -af`

    # echo "[INFO]Warm up " $1 " ..."
    # re=`time docker build -q --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .`

    for ((i=1; i<=${loop}; i++))do      
        while true;do
            echo "[INFO]Clean all docker images before test ..."
            re=`docker system prune -af` 
            echo "[INFO]Start " $1 "-" ${i} " default test ..."
            startTraffic1=`cat /proc/net/dev | grep enp | awk '{print $2}'`
            re=`time docker build -q -t test -f Dockerfile . `
            endTraffic1=`cat /proc/net/dev | grep enp | awk '{print $2}'`
            echo "[INFO]Test " $1 "-" ${i} ", default = " $((endTraffic1 - startTraffic1)) 
            if [[ $re =~ "sha256" ]];then
                echo $re
                break
            else
                echo "Error occurs in default test, possibly dur to unstable network, the image build will be restart ..."
            fi
        done
    done

    for ((i=1; i<=${loop}; i++))do      
        while true;do
            echo "[INFO]Clean all docker images before test ..."
            re=`docker system prune -af` 
            echo "[INFO]Start " $1 "-" ${i} " cached test ..."
            startTraffic2=`cat /proc/net/dev | grep enp | awk '{print $2}'`
            re=`time docker build -q --build-arg http_proxy=http://${host}:8080 --build-arg HTTP_PROXY=http://${host}:8080 -t test -f Dockerfile .`
            endTraffic2=`cat /proc/net/dev | grep enp | awk '{print $2}'`
            echo "[INFO]Test " $1 "-" ${i} ", cached = " $((endTraffic2 - startTraffic2)) 
            if [[ $re =~ "sha256" ]];then
                echo $re
                break
            else
                echo "Error occurs in cached test, possibly dur to unstable network, the image build will be restart ..."
            fi
        done
    done
}

for file in ${lists[@]};do
    echo ${file}
    cd ${file} # enter dir
    if [ -f "Dockerfile" ];then
        RUN ${file} ${loop}
    fi
    cd ..
done
