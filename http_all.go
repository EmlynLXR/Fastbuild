package main

import(
	"fmt"
	"net/http"
	//"net/url"
	//"errors"
	//"strings"
	//"path/filepath"
	"net"
	"sync"
	"io"
	"io/ioutil"
	"time"
	"bytes"
	"os"
	"strconv"
	"strings"
	"os/signal"
	"syscall"
)

type FileCached struct{
	FileName	string
	//LastUse	time.Time
	LastMod	time.Time
}

type FileDB struct{
	Files	map[string]*FileCached
}

const (
    seconds      = 1e11
    milliseconds = 1e14
    microseconds = 1e17
	RETRY_CHECKTIME = 10
)

var (
	fileDB FileDB 
	mu sync.RWMutex
	cachePath string = "cached/"
	cacheTime string = "cached/time.log"
	hostMacAddr net.HardwareAddr
	sigs chan os.Signal
	done chan bool
	fileType  = []string{".deb",".gz",".apk",".tar",".asc",".tgz",".sha256",".x86_64",".noarch",".xml",".rpm"}
)

func TimestampToTime(ts string) (time.Time) {
    i, err := strconv.ParseInt(ts, 10, 64)
    if err != nil {
        return time.Time{}
    }
    if i < seconds {
        return time.Unix(i, 0)
    }
    if i < milliseconds {
        return time.Unix(i/1000, (i%1000)*1e6)
    }
    if i < microseconds {
        return time.Unix(i/1e6, (i%1e6)*1000)
    }
    return time.Unix(0, i)
}

func handleIterceptor(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
		h(w, r)
    }
}

func SplitString(s string, myStrings []rune) []string {
	Split := func(r rune) bool {
		for _, v := range myStrings {
			if v == r {
				return true
			}
		}
		return false
	}
	a := strings.FieldsFunc(s, Split)
	return a
}

func InitFileDB(){
	fmt.Println("Init fileDB ... ")
	fileDB = FileDB{make(map[string]*FileCached)}
	if _, err := os.Stat(cacheTime); os.IsNotExist(err) {
		fmt.Println("Finist init fileDB without time.log ... ")
		return 
	}
    content, err := ioutil.ReadFile(cacheTime)
    if err != nil {
        panic(err)
    }
	slice := strings.Split(string(content),",")
	for _,value := range slice{
		tmp := strings.Split(value," ")
		if len(tmp) == 2 {

			if _, err := os.Stat(cachePath+tmp[0]); os.IsNotExist(err) {
				continue 
			}	

			fileDB.Files[tmp[0]] = &FileCached{
				FileName:	tmp[0],
				//LastUse:	TimestampToTime(tmp[1]),
				LastMod:	TimestampToTime(tmp[1]).UTC(),
			}
			fmt.Println(fileDB.Files[tmp[0]] )
		}
	}
	fmt.Println("Finist init fileDB with ", len(fileDB.Files) , " files ... ")
}

func SaveFileDB(){
	fmt.Println("Save fileDB ... ")
    file, err := os.OpenFile(cacheTime, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
    if err != nil {
        fmt.Println("Open time.log failed,err:", err)
        return
    }
    defer file.Close()
	for key, value := range fileDB.Files {
		//fmt.Println(value)
		//file.WriteString(key + " " + strconv.FormatInt(value.LastUse.UnixNano(),10) + " " + strconv.FormatInt(value.LastMod.UnixNano(),10) + ",")
		file.WriteString(key + " " + strconv.FormatInt(value.LastMod.UnixNano(),10) + ",")
	}
	fmt.Println("Finist save fileDB with ", len(fileDB.Files), " files ... ")
}

func getFilename(str string) (string){
	slices := SplitString(str, []rune{'&','=','/'})
	for _,strT := range slices{
		for _,value := range fileType{
			if strings.HasSuffix(strT,value) == true{
				return strT
			}
		}
	}

	if strings.HasPrefix(str, "http://github.com/") && strings.Contains(str, "releases/download/"){
		slices := SplitString(str, []rune{'&','=','/'})
		flag := 0
		filename := ""
		for _,strT := range slices{
			if flag == 1 {
				if filename == ""{
					filename = strT
				}else{
					filename =  strT + "_" + filename
				}
			}
			if strT == "download"{
				flag = 1
			}
		}
		return filename
	}

	// if strings.Contains(str, "/by-hash/SHA256/"){
	// 	slices := SplitString(str, []rune{'&','=','/'})
	// 	filename := slices[len(slices)-1] + "_sha256"
	// 	return filename,true
	// }
	return ""
}

func searchCache(filename string) (bool) {
	if _, err := os.Stat(cachePath+filename); os.IsNotExist(err) {
		return false 
	}else{
		return true
	}
}

func checkTime(urlReq string,filename string) (bool){
	var (
		chlocaltime = make(chan time.Time,1)
		chremotetime = make(chan time.Time,1)
	)

	// get local time
	go func(){
		mu.RLock()
		defer mu.RUnlock()
		file,ok := fileDB.Files[filename]
		if ok {
			chlocaltime <- file.LastMod
		}
	}()

	//get remote time
	go func(){
		for i := 0; i <= RETRY_CHECKTIME + 1; i++{
			if i > 0{
				fmt.Println("CheckTime will start the ", i , " retry ... ")
			}
			if i > RETRY_CHECKTIME{
				fmt.Println("Httpserver has tried HEAD requests for checkTime ", RETRY_CHECKTIME, " times , we will ignore the local cache in this situation")
				break;
			}
			res, err := http.Head(urlReq); 
			if err != nil{
				fmt.Println("Error occures at Head,", err, urlReq)
				continue
			}
			if LastMod := res.Header.Get("Last-Modified"); LastMod != ""{
				if parsedMTime, err := http.ParseTime(LastMod); err == nil {
					chremotetime <- parsedMTime
					break
				}else{
					fmt.Println("Error occures at ParseTime LastMod,", err, urlReq)
					res.Body.Close()
					continue
				}
			}else {
				fmt.Println("Error occures at get LastMod,", err, urlReq)
				res.Body.Close()
				continue
			}
			// if LastUse = res.Header.Get("Date"); LastUse != ""{
			// 	if parsedMTime, err := http.ParseTime(LastUse); err == nil {
			// 		LastUseTime = parsedMTime
			// 	}else{
			// 		fmt.Println("Error occures at ParseTime LastUse,", err, urlReq)
			// 		res.Body.Close()
			// 		continue
			// 	}
			// }else{
			// 	fmt.Println("Error occures at get LastUse,", err, urlReq)
			// 	res.Body.Close()
			// 	continue
			// }
		}
	}()

	LocalModTime := <- chlocaltime
	LastModTime := <- chremotetime
	return LastModTime.Equal(LocalModTime)
	// if LastModTime.Equal(LocalModTime) == false{
	// 	return errors.New(filename + " 's localModtime " + LocalModTime.String() + " is not equal with remoteModTime " + LastModTime.String())
	// 	//return LastModTime,LastUseTime,errors.New(filename + " 's localModtime " + LocalModTime.String() + " is not equal with remoteModTime " + LastModTime.String())
	// }else{
	// 	return nil
	// 	//return LastModTime,LastUseTime,nil
	// }
}

func makeNewReq(request *http.Request) (*http.Response, int, string, string){
	var (
		resp *http.Response
		err	error = nil
		urlReq string = request.URL.String() 
		method string = request.Method 
		filename string 
		// -1 not a deb file
		// 0 no need to get remote, we can use local file
		// 1 need to get remote, local file expires
		// 2 need to get remote, local file not exits
		REGET int = -1
		path string 
	)

	switch method {
		case "GET":
			if filename = getFilename(urlReq);filename != "" {
				path = cachePath + filename
				if _, err := os.Stat(cachePath+filename); os.IsNotExist(err) == false{
					//local file exit
					if checkTime(urlReq,filename) == true {
						REGET = 0
						// fmt.Println("Cache hit, Time hit, ",filename)
						// mu.Lock()
						// fileDB.Files[filename].LastUse = time.Now()
						// mu.Unlock()
					}else{
						REGET = 1
						// fmt.Println("Cache hit, Time miss",filename)
					}
				} else{
					// fmt.Println("Cache Miss, ",filename)
					REGET = 2
				}
			}
			// if isFile == false, then REGET = -1
			if REGET != 0{
				resp, err = http.Get(urlReq)
			}
		case "HEAD":
			resp, err = http.Head(urlReq)
		case "POST":
		case "PUT":
		case "PATCH":
		case "DELETE":
		case "CONNECT":
		case "OPTIONS":
		case "TRACE":
		default:
			fmt.Println("error in method mismatch, ",method,urlReq, err)
	}

	if err != nil {
		fmt.Println("error in ", method, ", ", urlReq, err)
		return nil,REGET,path,filename
	}	
	return resp,REGET,path,filename
}

func saveLocal(resp *http.Response, path string, filename string)(error){
	var (
		err error
		//LastUseTime time.Time
		LastModTime time.Time
	)
	//fmt.Println("Start saving file ", path, " at ",time.Now())
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("os.Create() err: %v.\n", err)
	}
	io.Copy(f, resp.Body)

	if LastMod := resp.Header.Get("Last-Modified");LastMod != "" {
		if parsedMTime, err := http.ParseTime(LastMod); err == nil {
			LastModTime = parsedMTime
		}
	}
	// if LastUse := resp.Header.Get("Date");LastUse != "" {
	// 	if parsedMTime, err := http.ParseTime(LastUse); err == nil {
	// 		LastUseTime = parsedMTime
	// 	}
	// }
	mu.Lock()
	if file,ok := fileDB.Files[filename] ; ok {
		file.LastMod = LastModTime
	}else{
		fileDB.Files[filename] = &FileCached{
			FileName:	filename,
			LastMod:	LastModTime,
			//LastUse:	LastUseTime,
		}
	}
	mu.Unlock()
	return err
}

func httpDefault(writer http.ResponseWriter, request *http.Request) {	
	// fmt.Println("[",time.Now(),"]Receive a http request, ",request.URL.String())

	// only rediect
	// var (
	// 	urlReq string = request.URL.String() 
	// 	method string = request.Method 
	// )
	// resp, err := http.Get(urlReq)
	// if err != nil {
	// 	fmt.Println("error in ", method, ", ", urlReq, err)
	// }else{
	// 	for key,value := range resp.Header{
	// 		writer.Header().Set(key,value[0])
	// 	}
	// 	writer.WriteHeader(resp.StatusCode)
	// 	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	// 	writer.Write(bodyBytes)	
	// }
	// return 

	resp,REGET,path,filename := makeNewReq(request)
	if resp != nil{
		defer resp.Body.Close()
	}

	// use local
	if REGET == 0 {
		fmt.Println("[", time.Now(), "]Local : ",filename)
		content, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("error in ReadFile, ", err)
			return 
		}
		writer.WriteHeader(http.StatusOK)
		writer.Write(content)
		return 
	}

	// use remote
	if filename != ""{
		fmt.Println("[", time.Now(), "]Remote : ",filename)
	}
	for key,value := range resp.Header{
		writer.Header().Set(key,value[0])
	}
	writer.WriteHeader(resp.StatusCode)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	writer.Write(bodyBytes)

	// save local
	if REGET > 0 {
		resp.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		go func(){
			if err := saveLocal(resp,path,filename);err != nil{
				fmt.Println("error in saveLocal, ",filename, err)
			}else{
				fmt.Println("[", time.Now(), "]Save : ",filename)
			}
		}()
	}
}

func main() {
	InitFileDB()
	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func(){
		defer close(sigs)
		sig := <-sigs
		fmt.Println("acquire signal:",sig)
		SaveFileDB()
		
	}()

	http.HandleFunc("/", handleIterceptor(httpDefault))
	http.ListenAndServe(":8080", nil)
}
