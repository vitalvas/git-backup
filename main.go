package main

import (
	"log"

	"github.com/vitalvas/git-backup/app/source"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	source.RunGitHub()
}