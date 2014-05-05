package main

import (
	"io/ioutil"
	"path"
	"fmt"
	"mime"
	"strings"
	//"os"
)

type SyncResourceInfo struct {
	Name string
	Path string
	Parent string
	IsDir bool
	MimeType string
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
	err, entries := syncer.BrowseFolder(folder)
	if err != nil{
		fmt.Println("Failed to browse entries ", err)
		return
	}
	//os.Exit(1)
	for _, file := range fileList{
		//Lookup for file and folder on the file system
		extensionSplit := strings.Split(file.Name(), ".")
		extension := extensionSplit[len(extensionSplit) - 1]
		//Extension needs to have the . otherwise it will fail
		mimeType := mime.TypeByExtension("." + extension)
		var entry = SyncResourceInfo{Name:file.Name(), Parent:folder, IsDir:file.IsDir(), Path:path.Join(real_path, file.Name()), MimeType:mimeType}
		if entry.IsDir{
			//Entry is a directory, create the directory remotely
			folder_path := entry.Parent
			if entry.Parent != "/"{
				//Special Case of the local file
				folder_path += "/"
			}
			folder_path += entry.Name
			if !syncer.HasFolder(folder_path){
				err = syncer.AddResource(&entry)
				if err != nil{
					fmt.Println("Failed to create Resource folder ", folder_path, " with error ", err)
					//Skip this folder, since we couldn't create the folder itself
					continue
				}
			}
			s.sync(syncer, real_path + "/" +entry.Name, folder_path)
		}else{
			var found = false
			for _, existing_entry := range entries{
				//fmt.Printf("Comparing %s with %s \n", existing_entry.Name, entry.Name)
				if existing_entry.Name == entry.Name{
					found = true
				}
			}
			if (!found){
				err = syncer.AddResource(&entry)
				if (err != nil){
					fmt.Println("Failed to add entry ", entry.Name, " with error ", err)
				}
			}
		}
	}
}