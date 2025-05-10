package resource_entity

import (
	"fmt"
	"path/filepath"
)

type Response struct {
	Path string `json:"path" example:"/folder1/folder2/"`
	Name string `json:"name" example:"folder2"`
	Size int64  `json:"size" example:"123456789"`
	Type string `json:"type" example:"DIRECTORY"`
} // @name Response

type Path struct {
	IsDirectory  bool
	OriginalPath string // requested path from client
	CleanPath    string // path to object without tailing /
}

// CleanPathWithTailingSlash returns current clean path with tailing /
func (p Path) CleanPathWithTailingSlash() string {
	return fmt.Sprintf("%s/", p.CleanPath)
}

// CleanPathDirName returns folder where this object exists
func (p Path) CleanPathDirName() string {
	if p.IsDirectory {
		return p.CleanPath
	}

	return filepath.Dir(p.CleanPath)
}
