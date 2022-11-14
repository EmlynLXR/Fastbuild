# fastbuild 
+ a daemon process and some child process in each containeer

### when docker daemon launches a container?
+ Fastbuild obtains the container's name and resolves the name to **get the id of the main process(init) in the container** by using the Docker API interface;
+ Fastbuild reads the process's **proc** file system to know the conatiner's Network namespace;
+ Forks child process and attaches it to the namespace by calling the **setns()** system all;
+ For request files, it then searches FastBuild's local cache to see whether it is in the cache;
+ If so, FastBuild **reads its last modification time** and generates a request to **get the file's last modification time at the server**;when two times match, the file will be retrieved from the cache and sent to the child process, which will deliver it to the process running the instructions in the conatiner by using **libpcap**;
+ If not,file is remotely retrieved as usual. After receiving the file, FastBuild daemon **stores it into the local cache**, possibly overwriting the out-of-date file in the cache, and delivers it via the child process.
