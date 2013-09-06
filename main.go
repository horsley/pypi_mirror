// pypi_mirror_client project main.go
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	UPSTREAM = "http://pypi.gocept.com" //上级Pypi源
	SAVEPATH = "/data/opensources/pypi" //本地存放目录
	ERRORLOG = "error.log"
	//SAVEPATH = "./test"

	PAGEIDX       = "/simple"
	PAGEIDX_SHARD = "/simple_shard"
	PAGEPKG       = "/packages"
	PAGESIG       = "/serversig"

	NUM_GOROUTINE = 500
	VERIFY_MD5    = true

	MAX_ERR_RETRY  = 5
	RETRY_INTERVAL = 500 * time.Millisecond
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	mirror()
}

func mirror() (err error) {
	var s string
	var total, count int

	errLog := make([]string, 0)
	defer func() {
		if len(errLog) != 0 { //写错误日志
			ioutil.WriteFile(ERRORLOG, []byte(strings.Join(errLog, "\n")), 0700)
		}
	}()

	fmt.Println("Building necessary directory structure...")
	buildDir()

	//下载总索引并提取各个包索引页链接
	fmt.Println("Downloading the main index...")
	links, _ := GetLinks(UPSTREAM+PAGEIDX, SAVEPATH+PAGEIDX+"/index.html")

	total = len(links)
	fmt.Println("Parse main index finished, total packages:", total)

	//准备并发开跑
	finish := make(chan [2]string, total)
	for i := 0; i < NUM_GOROUTINE; i++ {
		go func(finish chan [2]string, i, n int) {
			var err_retry int
			if n > total { //防止最后一次下标越界
				n = total
			}
			for ; i < n; i++ {
				var statusOut [2]string
				var count [4]int //记录一个包的文件总数、下载文件数、跳过文件数、错误文件数

				statusOut[0] = links[i].Name

				//下载包索引页，并提取包各版本文件链接
				//fix ext3 32k sub-dir problem by softlink
				realPath := SAVEPATH + PAGEIDX_SHARD + "/" + strings.ToLower(links[i].Name[:1]) + "/" + links[i].Name
				fakePath := SAVEPATH + PAGEIDX + "/" + links[i].Name
				pLinks, err := GetLinks(links[i].FullUrl, realPath+"/index.html")
				if r, err := os.Readlink(fakePath); err != nil || r != realPath {
					os.Remove(fakePath)
					os.Symlink(realPath, fakePath)
				}

				if err != nil {
					errLog = append(errLog, time.Now().String()+" fetch index error: "+links[i].FullUrl)
					break
				}

				//下载包签名证书
				for err_retry = 0; err_retry < MAX_ERR_RETRY; err_retry++ {
					if s, err = FetchAndSave(UPSTREAM+PAGESIG+"/"+links[i].Name, SAVEPATH+PAGESIG+"/"+links[i].Name, false); err == nil {
						break
					}
					time.Sleep(RETRY_INTERVAL)
				}
				if err != nil {
					errLog = append(errLog, time.Now().String()+" fetch signature error: "+UPSTREAM+PAGESIG+"/"+links[i].Name)
					break
				}

				//下载包的所有版本并校验
				for _, pkgFile := range pLinks {
					url := strings.Split(pkgFile.FullUrl, "#md5=")  //切分下载url和md5
					dir := strings.Split(url[0], PAGEPKG)           //获取本地保存路径
					if strings.HasPrefix(pkgFile.RawHref, "http") { //跳过外链
						continue
					}
					if len(url) == 1 || len(dir) == 1 {
						//有些包里面索引页会有包主页链接什么的 如http://e.pypi.python.org/simple/1ee/这里面
						//这时候跳过这个链接
						continue
					}
					count[0]++

					for err_retry = 0; err_retry < MAX_ERR_RETRY; err_retry++ {
						if s, err = FetchAndSave(url[0], SAVEPATH+PAGEPKG+dir[1], false); err == nil {
							break
						}
						time.Sleep(RETRY_INTERVAL)
					}
					if err != nil {
						errLog = append(errLog, time.Now().String()+" fetch package error: "+url[0])
					}
					switch s {
					case "ok":
						count[1]++
					case "skip":
						count[2]++
					case "fail":
						count[3]++
					}
					//@todo: md5 check
					runtime.Gosched()
				}
				statusOut[1] = fmt.Sprintf("total:%3d ok:%3d skip:%3d fail:%3d", count[0], count[1], count[2], count[3])

				finish <- statusOut
				runtime.Gosched()
			}
		}(finish, i*total/NUM_GOROUTINE, (i+1)*total/NUM_GOROUTINE)
	}
	fmt.Println("Start", NUM_GOROUTINE, "worker goroutine finish, wait for the result...")
	for i := 0; i < total; i++ {
		statusOut := <-finish
		count++
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("[%6d/%-6d] Package: %s\n", count, total, statusOut[0])
		fmt.Println(statusOut[1] + "\n")
	}

	fmt.Println("Finish!")
	return
}

func buildDir() {
	os.MkdirAll(SAVEPATH+PAGEIDX, 0755)
	os.MkdirAll(SAVEPATH+PAGEPKG, 0755)
	os.MkdirAll(SAVEPATH+PAGESIG, 0755)
}

func GetLinks(url, save string) (ret []Link, err error) {
	var tmp []byte
	var err_retry int

	for err_retry = 0; err_retry < MAX_ERR_RETRY; err_retry++ { //错误重试
		if _, err = FetchAndSave(url, save, false); err == nil {
			break
		}
		time.Sleep(RETRY_INTERVAL)
	}
	if err != nil {
		return
	}

	if tmp, err = ioutil.ReadFile(save); err != nil {
		panic(err)
	}
	links := linkRe.FindAllStringSubmatch(string(tmp), -1)

	ret = make([]Link, len(links))
	for k, i := range links {
		ret[k] = Link{i[2], url + "/" + i[1], i[1]}
	}
	return
}
