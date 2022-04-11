package backup

import (
	"log"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

func NewBackupRepo(path, cloneUrl string) {
	start := time.Now()

	fetchOpts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"+refs/*:refs/*"},
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Println("add new repo", cloneUrl)

		opts := &git.CloneOptions{
			URL: cloneUrl,
		}

		repo, err := git.PlainClone(path, true, opts)
		if err != nil {
			log.Fatal(err)
		}

		if err := repo.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
			log.Fatal(err)
		}

	} else {
		log.Println("updating repo", cloneUrl)

		repo, err := git.PlainOpen(path)
		if err != nil {
			log.Fatal(err)
		}

		if err := repo.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
			log.Fatal(err)
		}
	}

	t := time.Now()
	elapsed := t.Sub(start)

	if elapsed > time.Minute {
		log.Println("repo", cloneUrl, "processing time:", elapsed)
	}
}
