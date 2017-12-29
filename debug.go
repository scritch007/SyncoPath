package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/scritch007/go-simplejson"
)

type debugSyncPlugin struct {
	File         string
	storedStruct *simplejson.Json
	lock         sync.Mutex
}

func newDebugSyncPlugin(f string) (*debugSyncPlugin, error) {
	var a = new(debugSyncPlugin)
	if 0 == len(f) {
		fmt.Println("Please provide path to debug file")
		fmt.Scanln(&f)
	}
	a.File = f
	if _, err := os.Stat(a.File); os.IsNotExist(err) {
		f, err := os.Create(a.File)
		if err != nil {
			return nil, err
		}
		f.Close()
		j, err := simplejson.NewJson([]byte("{}"))
		res, _ := j.Encode()
		ioutil.WriteFile(a.File, res, 777)
	}
	file, err := ioutil.ReadFile(a.File)
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		return nil, err
	}
	a.storedStruct, err = simplejson.NewJson(file)
	if err != nil {
		fmt.Print("Couldn't deserialize file")
		return nil, err
	}
	return a, nil
}

// Name ...
func (p *debugSyncPlugin) Name() string {
	return "debugSyncPlugin"
}

// Lock log plugin
func (p *debugSyncPlugin) Lock() {
	DEBUG.Println("Locking")
	p.lock.Lock()
}

// Unlock unlock plugin
func (p *debugSyncPlugin) Unlock() {
	p.lock.Unlock()
	DEBUG.Println("Unlocked")
}

// BrowseFolder ..
func (p *debugSyncPlugin) BrowseFolder(folder string) ([]SyncResourceInfo, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var jsonFolder *simplejson.Json
	if folder == "/" {
		jsonFolder = p.storedStruct
	} else {
		splits := strings.Split(folder, "/")
		jsonFolder = p.storedStruct.GetPath(splits[1:]...)
		if jsonFolder == nil {
			return nil, errors.New("Folder doesn't exist")
		}
	}

	entries, err := jsonFolder.Map()
	if nil != err {
		DEBUG.Println(*jsonFolder, folder)
		return nil, errors.New("Wrong type")
	}
	var nbEntries = 0
	var result = make([]SyncResourceInfo, 10)
	for key := range entries {
		je := jsonFolder.Get(key)
		isDir := je.Get("IsDir").MustBool()
		if isDir {
			continue
		}
		result[nbEntries].Name = je.Get("Name").MustString()
		result[nbEntries].Parent = je.Get("Parent").MustString()
		result[nbEntries].Path = je.Get("Path").MustString()
		result[nbEntries].IsDir = isDir

		if nbEntries+2 > cap(result) {
			newSlice := make([]SyncResourceInfo, cap(result)*2)
			copy(newSlice, result)
			result = newSlice
		}
		nbEntries++
	}
	return result[:nbEntries], nil
}

// HasFolder ...
func (p *debugSyncPlugin) HasFolder(folder string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	splits := strings.Split(folder, "/")
	res := p.storedStruct.GetPath(splits[1:]...)
	return nil != res.MustMap()
}

// RemoveResource ...
func (p *debugSyncPlugin) RemoveResource(r SyncResourceInfo) error {
	return nil
}

// DownloadResource ...
func (p *debugSyncPlugin) DownloadResource(r *SyncResourceInfo) error {
	return nil
}

// AddResource ...
func (p *debugSyncPlugin) AddResource(r *SyncResourceInfo) error {
	DEBUG.Printf("Adding %s to path %s\n", r.Name, r.Parent)
	p.lock.Lock()
	defer p.lock.Unlock()

	j := p.storedStruct
	splits := strings.Split(r.Parent, "/")
	for _, path := range splits {
		if path == "" {
			continue
		}
		temp, found := j.CheckGet(path)
		if !found {
			return errors.New("Path doesn't exist")
		}
		j = temp
	}

	newEntry, err := json.Marshal(r)
	if err != nil {
		return err
	}
	newJSON, err := simplejson.NewJson(newEntry)

	j.Set(r.Name, *newJSON)
	if err != nil {
		fmt.Println("Couldn't call the Map")
		return err
	}
	res, err := p.storedStruct.EncodeIndent("", "  ")
	if err != nil {
		return err
	}
	ioutil.WriteFile(p.File, res, 777)
	return nil
}

// GetResourceInfo ...
func (p *debugSyncPlugin) GetResourceInfo(folder string) (SyncResourceInfo, error) {
	return SyncResourceInfo{}, nil
}

func (p *debugSyncPlugin) SyncOnlyMedia() bool {
	return false
}
