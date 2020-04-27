package oniontree

import "os"

func isDir(pth string) bool {
	f, err := os.Stat(pth)
	if err != nil {
		return false
	}
	if !f.Mode().IsDir() {
		return false
	}
	return true
}

func isFile(pth string) bool {
	f, err := os.Stat(pth)
	if err != nil {
		return false
	}
	if !f.Mode().IsRegular() {
		return false
	}
	return true
}
