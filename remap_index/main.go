// remap_index project main.go
/*
	本文件用于解决ext3文件系统上面32k一级子目录限制
	解决思路是把simple下各目录的索引文件夹移走按首字母分开存放
	在原来的位置做软连接到新的位置，这样对外暴露路径依然是兼容pep381的
*/
package main

import (
	"os"
	"path"
	"strings"
)

const (
	OLD_INDEX_DIR = "/data/opensources/pypi/simple"
	NEW_INDEX_DIR = "/data/opensources/pypi/simple_shard"
)

func main() {
	a, _ := readDir(OLD_INDEX_DIR)
	for _, pkgName := range a {
		if f, err := os.Stat(pkgName); err == nil && f.IsDir() {
			oldDir := OLD_INDEX_DIR + "/" + pkgName
			newDir := NEW_INDEX_DIR + "/" + strings.ToLower(pkgName[:1]) + "/" + pkgName

			os.MkdirAll(path.Dir(newDir), 0755)
			//保存到新目录的带首字母小写的子目录中
			os.RemoveAll(newDir)
			os.Rename(oldDir, newDir)
			//然后做软连接回去
			os.Symlink(newDir, oldDir)

		}
	}
}

func readDir(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	return list, nil
}
