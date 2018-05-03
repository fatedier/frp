package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Project contains name, license and paths to projects.
type Project struct {
	absPath string
	cmdPath string
	srcPath string
	license License
	name    string
}

// NewProject returns Project with specified project name.
func NewProject(projectName string) *Project {
	if projectName == "" {
		er("can't create project with blank name")
	}

	p := new(Project)
	p.name = projectName

	// 1. Find already created protect.
	p.absPath = findPackage(projectName)

	// 2. If there are no created project with this path, and user is in GOPATH,
	// then use GOPATH/src/projectName.
	if p.absPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			er(err)
		}
		for _, srcPath := range srcPaths {
			goPath := filepath.Dir(srcPath)
			if filepathHasPrefix(wd, goPath) {
				p.absPath = filepath.Join(srcPath, projectName)
				break
			}
		}
	}

	// 3. If user is not in GOPATH, then use (first GOPATH)/src/projectName.
	if p.absPath == "" {
		p.absPath = filepath.Join(srcPaths[0], projectName)
	}

	return p
}

// findPackage returns full path to existing go package in GOPATHs.
func findPackage(packageName string) string {
	if packageName == "" {
		return ""
	}

	for _, srcPath := range srcPaths {
		packagePath := filepath.Join(srcPath, packageName)
		if exists(packagePath) {
			return packagePath
		}
	}

	return ""
}

// NewProjectFromPath returns Project with specified absolute path to
// package.
func NewProjectFromPath(absPath string) *Project {
	if absPath == "" {
		er("can't create project: absPath can't be blank")
	}
	if !filepath.IsAbs(absPath) {
		er("can't create project: absPath is not absolute")
	}

	// If absPath is symlink, use its destination.
	fi, err := os.Lstat(absPath)
	if err != nil {
		er("can't read path info: " + err.Error())
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		path, err := os.Readlink(absPath)
		if err != nil {
			er("can't read the destination of symlink: " + err.Error())
		}
		absPath = path
	}

	p := new(Project)
	p.absPath = strings.TrimSuffix(absPath, findCmdDir(absPath))
	p.name = filepath.ToSlash(trimSrcPath(p.absPath, p.SrcPath()))
	return p
}

// trimSrcPath trims at the beginning of absPath the srcPath.
func trimSrcPath(absPath, srcPath string) string {
	relPath, err := filepath.Rel(srcPath, absPath)
	if err != nil {
		er(err)
	}
	return relPath
}

// License returns the License object of project.
func (p *Project) License() License {
	if p.license.Text == "" && p.license.Name != "None" {
		p.license = getLicense()
	}
	return p.license
}

// Name returns the name of project, e.g. "github.com/spf13/cobra"
func (p Project) Name() string {
	return p.name
}

// CmdPath returns absolute path to directory, where all commands are located.
func (p *Project) CmdPath() string {
	if p.absPath == "" {
		return ""
	}
	if p.cmdPath == "" {
		p.cmdPath = filepath.Join(p.absPath, findCmdDir(p.absPath))
	}
	return p.cmdPath
}

// findCmdDir checks if base of absPath is cmd dir and returns it or
// looks for existing cmd dir in absPath.
func findCmdDir(absPath string) string {
	if !exists(absPath) || isEmpty(absPath) {
		return "cmd"
	}

	if isCmdDir(absPath) {
		return filepath.Base(absPath)
	}

	files, _ := filepath.Glob(filepath.Join(absPath, "c*"))
	for _, file := range files {
		if isCmdDir(file) {
			return filepath.Base(file)
		}
	}

	return "cmd"
}

// isCmdDir checks if base of name is one of cmdDir.
func isCmdDir(name string) bool {
	name = filepath.Base(name)
	for _, cmdDir := range []string{"cmd", "cmds", "command", "commands"} {
		if name == cmdDir {
			return true
		}
	}
	return false
}

// AbsPath returns absolute path of project.
func (p Project) AbsPath() string {
	return p.absPath
}

// SrcPath returns absolute path to $GOPATH/src where project is located.
func (p *Project) SrcPath() string {
	if p.srcPath != "" {
		return p.srcPath
	}
	if p.absPath == "" {
		p.srcPath = srcPaths[0]
		return p.srcPath
	}

	for _, srcPath := range srcPaths {
		if filepathHasPrefix(p.absPath, srcPath) {
			p.srcPath = srcPath
			break
		}
	}

	return p.srcPath
}

func filepathHasPrefix(path string, prefix string) bool {
	if len(path) <= len(prefix) {
		return false
	}
	if runtime.GOOS == "windows" {
		// Paths in windows are case-insensitive.
		return strings.EqualFold(path[0:len(prefix)], prefix)
	}
	return path[0:len(prefix)] == prefix

}
