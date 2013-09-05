// pypi_mirror_client project main.go
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
)

const (
	UPSTREAM = "http://e.pypi.python.org" //上级Pypi源
	SAVEPATH = "/data/opensources/pypi"   //本地存放目录
	//SAVEPATH = "./test"

	PAGEIDX = "/simple"
	PAGEPKG = "/packages"
	PAGESIG = "/serversig"

	NUM_GOROUTINE = 50
	VERIFY_MD5    = true
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	mirror()
}

func mirror() {
	var total, count int

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
			if n > total { //防止最后一次下标越界
				n = total
			}
			for ; i < n; i++ {
				var statusOut [2]string

				statusOut[0] = links[i].Name
				//下载包索引页，并提取包各版本文件链接
				pLinks, _ := GetLinks(links[i].FullUrl, SAVEPATH+PAGEIDX+"/"+links[i].Name+"/index.html")
				//下载包签名证书
				FetchAndSave(UPSTREAM+PAGESIG+"/"+links[i].Name, SAVEPATH+PAGESIG+"/"+links[i].Name, false)

				//下载包的所有版本并校验
				for _, pkgFile := range pLinks {
					url := strings.Split(pkgFile.FullUrl, "#md5=") //切分下载url和md5
					dir := strings.Split(url[0], PAGEPKG)          //获取本地保存路径
					if len(url) == 1 || len(dir) == 1 {
						//有些包里面索引页会有包主页链接什么的 如http://e.pypi.python.org/simple/1ee/这里面
						//这时候跳过这个链接
						continue
					}
					s, _ := FetchAndSave(url[0], SAVEPATH+PAGEPKG+dir[1], false)
					statusOut[1] = statusOut[1] + pkgFile.Name + " [" + s + "]\n"

					//@todo: md5 check

				}

				finish <- statusOut
			}
		}(finish, i*total/NUM_GOROUTINE, (i+1)*total/NUM_GOROUTINE)
	}

	for i := 0; i < total; i++ {
		statusOut := <-finish
		count++
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println(" Package: "+statusOut[0], "[", count, "/", total, "]")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println(statusOut[1])
	}

	fmt.Println("Finish!")
}

func buildDir() {
	os.MkdirAll(SAVEPATH+PAGEIDX, 0700)
	os.MkdirAll(SAVEPATH+PAGEPKG, 0700)
	os.MkdirAll(SAVEPATH+PAGESIG, 0700)
}

func GetLinks(url, save string) (ret []Link, err error) {
	var tmp []byte
	FetchAndSave(url, save, true)

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
