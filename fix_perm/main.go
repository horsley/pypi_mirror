// fix_perm project main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && info.Mode() != 0755 {
			err := os.Chmod(path, 0755)
			fmt.Println("Change Dir", path, "perm 0755", err)
		} else {
			err := os.Chmod(path, 0644)
			fmt.Println("Change File", path, "perm 0644", err)
		}

		return nil
	})
}
