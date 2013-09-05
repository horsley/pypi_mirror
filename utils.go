// utils
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"
)

var linkRe = regexp.MustCompile(`(?m)<a.+?href=['"](.+?)['"].*?>(.+?)</a>`)

const MAX_ERR_RETRY = 5
const RETRY_INTERVAL = 500 * time.Millisecond

type Link struct {
	Name,
	FullUrl,
	RawHref string
}

type HttpError struct {
	err string
}

func (h *HttpError) Error() string {
	return h.err
}

func FetchAndSave(url, to string, force bool) (status string, err error) {
	var out *os.File
	var fileinfo os.FileInfo
	var resp *http.Response
	var err_retry int

	defer func() { //失败重试机制
		panic_err := recover()
		if panic_err != nil {
			if panic_err == "[HEAD ERR]" {

				for ; err_retry < MAX_ERR_RETRY; err_retry++ {
					time.Sleep(RETRY_INTERVAL)
					if resp, err = http.Get(url); err == nil { //没有出错则跳出重试过程
						continue
					}
				}
				if err != nil {
					panic("[HEAD ERR] Retry " + fmt.Sprintf("%v", MAX_ERR_RETRY) + " times, all failed!")
				}

			} else if panic_err == "[GET ERR]" {
				for ; err_retry < MAX_ERR_RETRY; err_retry++ {
					time.Sleep(RETRY_INTERVAL)
					if resp, err = http.Get(url); err == nil { //没有出错则跳出重试过程
						continue
					}
				}
				if err != nil {
					panic("[GET ERR] Retry" + fmt.Sprintf("%v", MAX_ERR_RETRY) + "times, all failed!")
				}

			} else {
				panic(panic_err)
			}
		}
	}()

	fileinfo, err = os.Stat(to)
	if err != nil {
		if os.IsNotExist(err) {
			//文件不存在，跳过检查
		} else {
			panic(err)
		}

	} else {
		if !force { //如果不是强制重新下载，则检验修改时间
			if resp, err = http.Head(url); err != nil {
				panic("[HEAD ERR]")
			}

			last_mod, _ := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
			if fileinfo.ModTime().After(last_mod) {
				status = "skip"
				return
			}
		}
	}

	os.MkdirAll(path.Dir(to), 0700) //建立层级的文件夹
	if out, err = os.Create(to); err != nil {
		panic(err)
	}
	defer out.Close()

	if resp, err = http.Get(url); err != nil {
		panic("[GET ERR]")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		status = resp.Status
		return
	}
	if _, err = io.Copy(out, resp.Body); err != nil {
		panic(err)
	}
	status = "ok"
	return
}
