package source

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/vitalvas/git-backup/app/backup"
	"golang.org/x/oauth2"
)

type GitHubSource struct {
	ctx         context.Context
	client      *github.Client
	httpClient  *http.Client
	user        string
	accessToken string
}

func NewGitHub() *GitHubSource {
	this := &GitHubSource{
		ctx:  context.Background(),
		user: os.Getenv("GITHUB_USER"),
	}

	this.accessToken = os.Getenv("GITHUB_TOKEN")

	this.httpClient = oauth2.NewClient(this.ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: this.accessToken},
	))

	this.client = github.NewClient(this.httpClient)

	return this
}

func (ghs *GitHubSource) Run() {
	_, response, err := ghs.client.Users.Get(ghs.ctx, ghs.user)
	if err != nil {
		log.Fatal(err)
	}

	if response.Rate.Remaining < 10 {
		delayTime := response.Rate.Reset.UTC().Sub(time.Now().UTC()) + (5 * time.Minute)

		log.Println("API rate limit expended. Used", response.Rate.Remaining, "of", response.Rate.Limit, ". Delay", delayTime)

		time.Sleep(delayTime)
	}

	var countRepos uint64

	if len(os.Getenv("GITHUB_SKIP_MAIN")) == 0 {
		countRepos += ghs.runUserRepos()
	}

	if len(os.Getenv("GITHUB_STARRED")) > 0 {
		countRepos += ghs.runUserStarred()
	}

	if len(os.Getenv("GITHUB_GIST")) > 0 {
		countRepos += ghs.runGist()
	}

	log.Println("total count", countRepos)
}

func (ghs *GitHubSource) runUserRepos() (count uint64) {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := ghs.client.Repositories.List(ghs.ctx, ghs.user, opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			if len(os.Getenv("GITHUB_SKIP_USER_FORKS")) > 0 && repo.GetFork() {
				continue
			}

			if ghs.backupRepo(repo) {
				count++
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return
}

func (ghs *GitHubSource) runUserStarred() (count uint64) {
	opts := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := ghs.client.Activity.ListStarred(ghs.ctx, ghs.user, opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			if ghs.backupRepo(repo.Repository) {
				count++
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return
}

func (ghs *GitHubSource) runGist() (count uint64) {
	opts := &github.GistListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		gists, resp, err := ghs.client.Gists.List(ghs.ctx, ghs.user, opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, gist := range gists {
			if !gist.GetPublic() {
				continue
			}

			u, err := url.Parse(gist.GetGitPullURL())
			if err != nil {
				log.Fatal(err)
			}

			storagePath := path.Join(os.Getenv("DATA_DIR"), "data", u.Host, u.Path)

			if err := backup.NewBackupRepo(storagePath, gist.GetGitPullURL(), true, nil); err != nil {
				log.Fatal(err)
			}

			count++
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return 0
}

func (ghs *GitHubSource) backupRepo(repo *github.Repository) bool {
	if len(repo.GetCloneURL()) == 0 {
		return false
	}

	u, err := url.Parse(repo.GetCloneURL())
	if err != nil {
		log.Fatal(err)
	}

	storagePath := path.Join(os.Getenv("DATA_DIR"), "data", u.Host,
		fmt.Sprintf("%s-%d.git", strings.TrimSuffix(u.Path, ".git"), repo.GetID()),
	)

	if repo.GetPrivate() {
		if err := backup.NewBackupRepo(storagePath, repo.GetCloneURL(), false, &ghs.accessToken); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := backup.NewBackupRepo(storagePath, repo.GetCloneURL(), false, nil); err != nil {
			log.Fatal(err)
		}
	}

	if repo.GetHasWiki() {
		wikiCloneURL := fmt.Sprintf("%s.wiki.git", strings.TrimSuffix(repo.GetCloneURL(), ".git"))

		storagePathWiki := path.Join(os.Getenv("DATA_DIR"), "data", u.Host,
			fmt.Sprintf("%s-%d.wiki.git", strings.TrimSuffix(u.Path, ".git"), repo.GetID()),
		)

		if repo.GetPrivate() {
			if err := backup.NewBackupRepo(storagePathWiki, wikiCloneURL, true, &ghs.accessToken); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := backup.NewBackupRepo(storagePathWiki, wikiCloneURL, true, nil); err != nil {
				log.Fatal(err)
			}
		}
	}

	return true
}
