package backup

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func NewBackupRepo(path, cloneUrl string, skipError bool, accessToken *string) {
	start := time.Now()

	fetchOpts := &git.FetchOptions{
		RefSpecs: []config.RefSpec{"+refs/*:refs/*"},
		Tags:     git.AllTags,
		Force:    true,
	}

	if accessToken != nil {
		fetchOpts.Auth = &http.BasicAuth{
			Username: "git",
			Password: *accessToken,
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		opts := &git.CloneOptions{
			URL:  cloneUrl,
			Auth: fetchOpts.Auth,
			Tags: git.AllTags,
		}

		repo, err := git.PlainClone(path, true, opts)

		if err != nil && !skipError && err != transport.ErrEmptyRemoteRepository {
			log.Fatal(err)
		} else if err != nil && skipError {
			return
		}

		log.Println("add new repo", cloneUrl)

		if repo != nil {
			if err := repo.Fetch(fetchOpts); err != nil &&
				err != git.NoErrAlreadyUpToDate &&
				err != git.ErrRemoteNotFound {
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
				if !strings.Contains(err.Error(), "ERR access denied or repository not exported") {
					log.Fatal(err)
				}
			}
		}
	}

	elapsed := time.Now().Sub(start)

	if elapsed > time.Minute {
		log.Println("repo", cloneUrl, "processing time:", elapsed)
	}
}
