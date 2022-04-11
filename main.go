package main

import (
	"log"
	"os"
	"time"

	"github.com/vitalvas/git-backup/app/api"
	"github.com/vitalvas/git-backup/app/source"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	if len(os.Getenv("API_SERVER_ADDR")) > 0 {
		go api.RunAPIServer()
	}

	github := source.NewGitHub()

	for {
		github.Run()

		time.Sleep(time.Hour)
	}
}
