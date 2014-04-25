package main

import (
  "io/ioutil"
  "fmt"
)

type Syncer struct{
  localFolder string
}

func NewSyncer(localFolder string) *Syncer{
  var syncer = new(Syncer)
  syncer.localFolder = localFolder
  return syncer
}

func (s *Syncer)Sync(){
  fileList, err := ioutil.ReadDir(s.localFolder)
  if (nil != err){
    fmt.Print("Something went wrong ", err)
  }
  for _, file := range fileList{
    if (file.IsDir()){
      //Check if repository exists in the "Cloud"
      fmt.Print(file.Name(), "\n")
    }
  }
}