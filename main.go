package main

import (
	"log"
	"time"

	"github.com/vitalvas/git-backup/app/source"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	for {
		source.RunGitHub()
		time.Sleep(time.Hour)
	}
}
