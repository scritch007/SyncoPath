package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/scritch007/go-simplejson"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type DebugSyncPlugin struct {
	file         string
	storedStruct *simplejson.Json
	lock         sync.Mutex
}

func NewDebugSyncPlugin(f string) (*DebugSyncPlugin, error) {
	var a = new(DebugSyncPlugin)
	a.file = f
	if _, err := os.Stat(a.file); os.IsNotExist(err) {
		f, err := os.Create(a.file)
		if err != nil {
			return nil, err
		}
		f.Close()
		j, err := simplejson.NewJson([]byte("{}"))
		res, _ := j.Encode()
		ioutil.WriteFile(a.file, res, 777)
	}
	file, err := ioutil.ReadFile(a.file)
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

func (p *DebugSyncPlugin) Name() string {
	return "DebugSyncPlugin"
}

func (p *DebugSyncPlugin) Lock() {
	DEBUG.Println("Locking")
	p.lock.Lock()
}

func (p *DebugSyncPlugin) Unlock() {
	p.lock.Unlock()
	DEBUG.Println("Unlocked")
}

func (p *DebugSyncPlugin) BrowseFolder(folder string) (error, []SyncResourceInfo) {
	p.lock.Lock()
	defer p.lock.Unlock()

	var jsonFolder *simplejson.Json
	if folder == "/" {
		jsonFolder = p.storedStruct
	} else {
		splits := strings.Split(folder, "/")
		jsonFolder = p.storedStruct.GetPath(splits[1:]...)
		if jsonFolder == nil {
			return errors.New("Folder doesn't exist"), nil
		}
	}

	entries, err := jsonFolder.Map()
	if nil != err {
		DEBUG.Println(*jsonFolder, folder)
		return errors.New("Wrong type"), nil
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
		nbEntries += 1
	}
	return nil, result[:nbEntries]
}

func (p *DebugSyncPlugin) HasFolder(folder string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	splits := strings.Split(folder, "/")
	res := p.storedStruct.GetPath(splits[1:]...)
	return nil != res.MustMap()
}

func (p *DebugSyncPlugin) RemoveResource(r SyncResourceInfo) error {
	return nil
}

func (p *DebugSyncPlugin) AddResource(r *SyncResourceInfo) error {
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
	newJson, err := simplejson.NewJson(newEntry)

	j.Set(r.Name, *newJson)
	if err != nil {
		fmt.Println("Couldn't call the Map")
		return err
	}
	res, err := p.storedStruct.EncodeIndent("", "  ")
	if err != nil {
		return err
	}
	ioutil.WriteFile(p.file, res, 777)
	return nil
}
