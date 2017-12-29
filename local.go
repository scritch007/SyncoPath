package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"

	simplejson "github.com/scritch007/go-simplejson"
)

// LocalSyncPlugin ...
type LocalSyncPlugin struct {
	Chroot string
}

// Name ...
func (l *LocalSyncPlugin) Name() string {
	return "Local"
}

// NewLocalSyncPlugin ...
func NewLocalSyncPlugin(config string) (*LocalSyncPlugin, error) {
	l := new(LocalSyncPlugin)
	if 0 == len(config) {
		fmt.Println("Please enter local path")
		fmt.Scanln(&l.Chroot)
	} else {
		token, err := simplejson.NewJson([]byte(config))
		if nil != err {
			return nil, err
		}
		tmp := token.Get("Chroot").MustString()
		l.Chroot = tmp
	}
	DEBUG.Printf("Local plugin chroot is %s", l.Chroot)

	return l, nil
}

// BrowseFolder ...
func (l *LocalSyncPlugin) BrowseFolder(f string) ([]SyncResourceInfo, error) {
	realPath := filepath.Join(l.Chroot, f)
	fileList, err := ioutil.ReadDir(realPath)

	DEBUG.Printf("Browsing %s which is actually %s", f, realPath)

	if nil != err {
		return nil, err
	}
	res := make([]SyncResourceInfo, len(fileList))
	for i, file := range fileList {
		res[i].IsDir = file.IsDir()
		res[i].Name = file.Name()
		res[i].Parent = f
		res[i].Path = filepath.Join(realPath, file.Name())
		res[i].MimeType = mime.TypeByExtension("." + filepath.Ext(file.Name()))
	}
	return res, nil
}

// RemoveResource ...
func (l *LocalSyncPlugin) RemoveResource(r SyncResourceInfo) error {
	return nil
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// AddResource ...
func (l *LocalSyncPlugin) AddResource(r *SyncResourceInfo) error {
	rPath := filepath.Join(l.Chroot, r.Parent, r.Name)
	if r.IsDir {
		err := os.Mkdir(rPath, os.ModePerm)
		if nil != err {
			return nil
		} else if os.IsExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		return copyFileContents(r.Path, rPath)
	}
}

func (l *LocalSyncPlugin) realPath(f string) string {
	return filepath.Join(l.Chroot, f)
}

// HasFolder ...
func (l *LocalSyncPlugin) HasFolder(f string) bool {
	DEBUG.Printf("Looking if %s exists. Real path is %s", f, l.realPath(f))
	if _, err := os.Stat(l.realPath(f)); os.IsNotExist(err) {
		return false
	}
	return true
}

// DownloadResource ...
func (l *LocalSyncPlugin) DownloadResource(r *SyncResourceInfo) error {
	//Do not read the information in the Path we will replace it with current file path
	r.Path = filepath.Join(l.Chroot, r.Parent, r.Name)
	return nil
}

// GetResourceInfo ...
func (l *LocalSyncPlugin) GetResourceInfo(folder string) (SyncResourceInfo, error) {
	var name, parent string
	if folder == "/" {
		name = "/"
		parent = "/"
	} else {
		name = filepath.Base(folder)
		parent = filepath.Dir(folder)
	}
	s := SyncResourceInfo{Name: name, Parent: parent, Path: filepath.Join(l.Chroot, folder), IsDir: true}
	return s, nil
}

func (l *LocalSyncPlugin) SyncOnlyMedia() bool {
	return false
}
