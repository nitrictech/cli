package utils

import (
	"io/fs"
	"time"
)

type stringFileInfo struct {
	name string
	size int64
}

func NewStringFileInfo(fileName, content string) fs.FileInfo {
	return &stringFileInfo{name: fileName, size: int64(len(content))}
}

func (i *stringFileInfo) Name() string       { return i.name }
func (i *stringFileInfo) Size() int64        { return i.size }
func (i *stringFileInfo) Mode() fs.FileMode  { return 0 }
func (i *stringFileInfo) ModTime() time.Time { return time.Now() }
func (i *stringFileInfo) IsDir() bool        { return false }
func (i *stringFileInfo) Sys() interface{}   { return nil }
