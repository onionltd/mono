package oniontree

import (
	"io"
	"os"
)

func isDir(pth string) bool {
	f, err := os.Stat(pth)
	if err != nil {
		return false
	}
	return f.Mode().IsDir()
}

func isEmptyDir(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}
	return false // Either not empty or error, suits both cases
}

func isFile(pth string) bool {
	f, err := os.Stat(pth)
	if err != nil {
		return false
	}
	return f.Mode().IsRegular()
}

func isSymlink(pth string) bool {
	f, err := os.Lstat(pth)
	if err != nil {
		return false
	}
	return f.Mode()&os.ModeSymlink != 0
}
