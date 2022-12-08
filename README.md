# fastbuild 
+ A daemon process and some child process in each containeer

## Solution in the paper
### when docker daemon launches a container?
+ Fastbuild obtains the container's name and resolves the name to **get the id of the main process(init) in the container** by using the Docker API interface;
+ Fastbuild reads the process's **proc** file system to know the conatiner's Network namespace;
+ Forks child process and attaches it to the namespace by calling the **setns()** system all;
+ For request files, it then searches FastBuild's local cache to see whether it is in the cache;
+ If so, FastBuild **reads its last modification time** and generates a request to **get the file's last modification time at the server**;when two times match, the file will be retrieved from the cache and sent to the child process, which will deliver it to the process running the instructions in the conatiner by using **libpcap**;
+ If not,file is remotely retrieved as usual. After receiving the file, FastBuild daemon **stores it into the local cache**, possibly overwriting the out-of-date file in the cache, and delivers it via the child process.

### Problem
+ How to inject return packet information?

## Actual solution
### Implement by net/http

## Test
### Version conflict issues, count 2+2+4+5+4+3+3+2+4+2=31
1. nginx：latest = 1.23.2 = 1.23; 1.22.1 = 1.22
2. bitnami/kafka：latest = 3.3.1 ＝ 3.3；3.2.3 = 3.2
3. zookeeper：latest；3.8.0；3.7.1 = 3.7；3.6.3
4. storm : latest；2.4.0；2.3.0；2.2.0；1.2.3
5. ubuntu：latest = 22.04；22.10；20.04；18.04
6. flink：latest = 1.16.0 ＝ java11；1.15.2（1.15.3回退）；java8
7. bitnami/spark：latest = 3.3.1；3.2.3 = 3.2；3.2.2（3.2.3回退）
8. redis：latest = 7.0.5 = 7.0 ；6.2.7 = 6.2
9. debian：latest = 11.5；10.13；stable-slim；stable-backports
10. openjdk：oraclelinux8 = jdk_oraclelinux8 = oracle = latest；oraclelinux7
### Usage
+ go run http_simple.go
+ bash test_all.sh
