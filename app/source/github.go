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

func (this *GitHubSource) Run() {
	_, response, err := this.client.Users.Get(this.ctx, this.user)
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
		countRepos += this.runUserRepos()
	}

	if len(os.Getenv("GITHUB_STARRED")) > 0 {
		countRepos += this.runUserStarred()
	}

	if len(os.Getenv("GITHUB_GIST")) > 0 {
		countRepos += this.runGist()
	}

	log.Println("total count", countRepos)
}

func (this *GitHubSource) runUserRepos() (count uint64) {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := this.client.Repositories.List(this.ctx, this.user, opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			if len(os.Getenv("GITHUB_SKIP_USER_FORKS")) > 0 && repo.GetFork() {
				continue
			}

			if this.backupRepo(repo) {
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

func (this *GitHubSource) runUserStarred() (count uint64) {
	opts := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := this.client.Activity.ListStarred(this.ctx, this.user, opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			if this.backupRepo(repo.Repository) {
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

func (this *GitHubSource) runGist() (count uint64) {
	opts := &github.GistListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		gists, resp, err := this.client.Gists.List(this.ctx, this.user, opts)
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

			backup.NewBackupRepo(storagePath, gist.GetGitPullURL(), true, nil)

			count++
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return
}

func (this *GitHubSource) backupRepo(repo *github.Repository) bool {
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
		backup.NewBackupRepo(storagePath, repo.GetCloneURL(), false, &this.accessToken)
	} else {
		backup.NewBackupRepo(storagePath, repo.GetCloneURL(), false, nil)
	}

	if repo.GetHasWiki() {
		wikiCloneURL := fmt.Sprintf("%s.wiki.git", strings.TrimSuffix(repo.GetCloneURL(), ".git"))

		storagePathWiki := path.Join(os.Getenv("DATA_DIR"), "data", u.Host,
			fmt.Sprintf("%s-%d.wiki.git", strings.TrimSuffix(u.Path, ".git"), repo.GetID()),
		)

		if repo.GetPrivate() {
			backup.NewBackupRepo(storagePathWiki, wikiCloneURL, true, &this.accessToken)
		} else {
			backup.NewBackupRepo(storagePathWiki, wikiCloneURL, true, nil)
		}
	}

	return true
}
