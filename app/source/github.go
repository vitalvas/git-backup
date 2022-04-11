package source

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/vitalvas/git-backup/app/backup"
	"golang.org/x/oauth2"
)

func RunGitHub() {
	ctx := context.Background()

	tc := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	))

	client := github.NewClient(tc)

	_, response, err := client.Users.Get(ctx, os.Getenv("GITHUB_USER"))
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
		countRepos += githubUserRepos(ctx, client)
	}

	if len(os.Getenv("GITHUB_STARRED")) > 0 {
		countRepos += githubUserStarred(ctx, client)
	}

	if len(os.Getenv("GITHUB_GIST")) > 0 {
		countRepos += githubGist(ctx, client)
	}

	log.Println("total count", countRepos)
}

func githubUserRepos(ctx context.Context, client *github.Client) (count uint64) {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.List(ctx, os.Getenv("GITHUB_USER"), opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			if githubBackupRepo(repo) {
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

func githubUserStarred(ctx context.Context, client *github.Client) (count uint64) {
	opts := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Activity.ListStarred(ctx, os.Getenv("GITHUB_USER"), opts)
		if err != nil {
			log.Fatal(err)
		}

		for _, repo := range repos {
			if githubBackupRepo(repo.Repository) {
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

func githubGist(ctx context.Context, client *github.Client) (count uint64) {
	opts := &github.GistListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		gists, resp, err := client.Gists.List(ctx, os.Getenv("GITHUB_USER"), opts)
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

			storagePath := path.Join("tmp", "data", u.Host, u.Path)

			backup.NewBackupRepo(storagePath, gist.GetGitPullURL(), true)

			count++
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return
}

func githubBackupRepo(repo *github.Repository) bool {
	if len(repo.GetCloneURL()) == 0 || repo.GetPrivate() {
		return false
	}

	u, err := url.Parse(repo.GetCloneURL())
	if err != nil {
		log.Fatal(err)
	}

	storagePath := path.Join("tmp", "data", u.Host,
		fmt.Sprintf("%s-%d.git", strings.TrimSuffix(u.Path, ".git"), repo.GetID()),
	)

	backup.NewBackupRepo(storagePath, repo.GetCloneURL(), false)

	if repo.GetHasWiki() {
		wikiCloneURL := fmt.Sprintf("%s.wiki.git", strings.TrimSuffix(repo.GetCloneURL(), ".git"))

		storagePathWiki := path.Join("tmp", "data", u.Host,
			fmt.Sprintf("%s-%d.wiki.git", strings.TrimSuffix(u.Path, ".git"), repo.GetID()),
		)

		backup.NewBackupRepo(storagePathWiki, wikiCloneURL, true)
	}

	return true
}
