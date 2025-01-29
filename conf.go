package main

import (
  "fmt"
  "os"
)

func main() {
  homeDir, err := os.UserHomeDir()
  if err != nil {
    return
  }
  if fileExists(homeDir+"/.gosh_history") {
    fmt.Println("GoSh History file exists")
  } else {
    fmt.Println("GoSh History file does not exist")
  }
}

func fileExists(filename string) bool {
   info, err := os.Stat(filename)
   if os.IsNotExist(err) {
      return false
   }
   return !info.IsDir()
}
