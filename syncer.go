package main

import (
	"io/ioutil"
	"fmt"
)

type SyncResourceInfo struct {
	Name string
	Path string
	Parent string
	IsDir bool
}

type SyncPlugin interface{
	Name() string
	BrowseFolder(f string) (error, []SyncResourceInfo)
	RemoveResource(r SyncResourceInfo) error
	AddResource(r *SyncResourceInfo) error
	HasFolder(folder string) bool
}

type Syncer struct{
	localFolder string
}

func NewSyncer(localFolder string) *Syncer{
	var syncer = new(Syncer)
	syncer.localFolder = localFolder
	return syncer
}

func (s *Syncer)Sync(syncer SyncPlugin){
	fmt.Println("Syncing with", syncer.Name())
	s.sync(syncer, s.localFolder, "/")
}
func (s *Syncer)sync(syncer SyncPlugin, real_path string, folder string){
	fileList, err := ioutil.ReadDir(real_path)
	if (nil != err){
		fmt.Print("Something went wrong ", err)
	}
	_, entries := syncer.BrowseFolder(folder)
	for _, file := range fileList{
		var entry = SyncResourceInfo{Name:file.Name(), Parent:folder, IsDir:file.IsDir()}
		if entry.IsDir{
			folder_path := entry.Parent
			if entry.Parent != "/"{
				folder_path += "/"
			}
			folder_path += entry.Name
			if !syncer.HasFolder(folder_path){
				syncer.AddResource(&entry)
			}
			s.sync(syncer, real_path + "/" +entry.Name, folder_path)
		}else{
			var found = false
			for _, existing_entry := range entries{
				if existing_entry.Name == entry.Name{
					found = true
				}
			}
			if (!found){
				syncer.AddResource(&entry)
			}
		}
	}
}