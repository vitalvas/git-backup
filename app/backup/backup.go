package backup

import (
	"log"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
)

func NewBackupRepo(path, cloneUrl string, skipError bool) {
	start := time.Now()

	fetchOpts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"+refs/*:refs/*"},
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		opts := &git.CloneOptions{
			URL: cloneUrl,
		}

		repo, err := git.PlainClone(path, true, opts)
		if err != nil && !skipError {
			log.Fatal(err)
		} else if err != nil && skipError {
			return
		}

		log.Println("add new repo", cloneUrl)

		if repo != nil {
			if err := repo.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
				log.Fatal(err)
			}
		}

	} else {
		log.Println("updating repo", cloneUrl)

		repo, err := git.PlainOpen(path)
		if err != nil {
			log.Fatal(err)
		}

		if repo != nil {
			if err := repo.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
				log.Fatal(err)
			}
		}
	}

	elapsed := time.Now().Sub(start)

	if elapsed > time.Minute {
		log.Println("repo", cloneUrl, "processing time:", elapsed)
	}
}
