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
	github := source.NewGitHub()

	for {
		github.Run()

		time.Sleep(time.Hour)
	}
}
