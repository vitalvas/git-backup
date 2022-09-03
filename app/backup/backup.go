package backup

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

var fileCache *cache.ObjectLRU

func init() {
	fileCache = cache.NewObjectLRU(64 * cache.MiByte)
}

func NewBackupRepo(path, cloneURL string, skipError bool, accessToken *string) error {
	start := time.Now()

	defer fileCache.Clear()

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
			URL:  cloneURL,
			Auth: fetchOpts.Auth,
			Tags: git.AllTags,
		}

		storage := filesystem.NewStorage(osfs.New(path), fileCache)
		defer storage.Close()

		repo, err := git.Clone(storage, nil, opts)

		if err != nil && !skipError && err != transport.ErrEmptyRemoteRepository {
			return err
		} else if err != nil && skipError {
			return nil
		}

		log.Println("add new repo", cloneURL)

		if repo != nil {
			if err := repo.Fetch(fetchOpts); err != nil &&
				err != git.NoErrAlreadyUpToDate &&
				err != git.ErrRemoteNotFound &&
				err != transport.ErrEmptyRemoteRepository {
				return err
			}
		}

	} else {
		log.Println("updating repo", cloneURL)

		repo, err := git.PlainOpen(path)
		if err != nil {
			return err
		}

		if repo != nil {
			if err := repo.Fetch(fetchOpts); err != nil && err != git.NoErrAlreadyUpToDate {
				if !strings.Contains(err.Error(), "ERR access denied or repository not exported") &&
					err != transport.ErrEmptyRemoteRepository {
					return err
				}
			}
		}
	}

	elapsed := time.Since(start)

	if elapsed > time.Minute {
		log.Println("repo", cloneURL, "processing time:", elapsed)
	}

	return nil
}
