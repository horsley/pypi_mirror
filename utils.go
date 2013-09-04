// utils
package main

import (
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"
)

var linkRe = regexp.MustCompile(`(?m)<a.+?href=['"](.+?)['"].*?>(.+?)</a>`)

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
				panic(err)
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
		panic(err)
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
