package source

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

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

	var countRepos uint64

	countRepos += githubUserRepos(ctx, client)

	if len(os.Getenv("GITHUB_STARRED")) > 0 {
		countRepos += githubUserStarred(ctx, client)
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
