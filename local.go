package main

import (
	"fmt"
	"github.com/scritch007/go-simplejson"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
)

type LocalSyncPlugin struct {
	Chroot string
}

func (l *LocalSyncPlugin) Name() string {
	return "Local"
}

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
		l.Chroot = token.Get("Chroot").MustString()
	}

	return l, nil
}

func (l *LocalSyncPlugin) BrowseFolder(f string) (error, []SyncResourceInfo) {
	real_path := filepath.Join(l.Chroot, f)
	fileList, err := ioutil.ReadDir(real_path)

	if nil != err {
		return err, nil
	}
	res := make([]SyncResourceInfo, len(fileList))
	for i, file := range fileList {
		res[i].IsDir = file.IsDir()
		res[i].Name = file.Name()
		res[i].Parent = f
		res[i].Path = filepath.Join(real_path, file.Name())
		res[i].MimeType = mime.TypeByExtension("." + filepath.Ext(file.Name()))
	}
	return nil, res
}
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

func (l *LocalSyncPlugin) AddResource(r *SyncResourceInfo) error {
	rPath := filepath.Join(l.Chroot, r.Parent, r.Name)
	return copyFileContents(r.Path, rPath)
}
func (l *LocalSyncPlugin) HasFolder(folder string) bool {
	if _, err := os.Stat(filepath.Join(l.Chroot, folder)); os.IsNotExist(err) {
		return false
	}
	return true
}

func (l *LocalSyncPlugin) DownloadResource(r *SyncResourceInfo) error {
	//Do not read the information in the Path we will replace it with current file path
	r.Path = filepath.Join(l.Chroot, r.Parent, r.Name)
	return nil
}

func (l *LocalSyncPlugin) GetResourceInfo(folder string) (error, SyncResourceInfo) {
	var name, parent string
	if folder == "/" {
		name = "/"
		parent = "/"
	} else {
		name = filepath.Base(folder)
		parent = filepath.Dir(folder)
	}
	s := SyncResourceInfo{Name: name, Parent: parent, Path: filepath.Join(l.Chroot, folder), IsDir: true}
	return nil, s
}
