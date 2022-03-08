import logging
import os.path
import requests
import shutil
import git
from datetime import datetime

from urllib.parse import urlparse
from uritemplate import expand as uri_expand


class GitBackup:
    def __init__(self):
        stream_handler = logging.StreamHandler()
        stream_handler.setLevel(logging.INFO)
        stream_handler.setFormatter(
            logging.Formatter('%(asctime)s - %(message)s')
        )

        self.log = logging.getLogger(__name__)
        self.log.setLevel('INFO')
        self.log.addHandler(stream_handler)

        self.session = requests.Session()
        self.repos = []
        self.dest_dir = os.getenv('DATA_DIR', '../data')
        self.archive_dir = os.getenv('ARCHIVE_DIR', os.path.join(self.dest_dir, 'archive'))

    def run(self):
        self.get_repos()
        self.make_mirror()

    def add_repo(self, id, clone_url, url):
        self.repos.append({
            'id': id,
            'clone_url': clone_url,
            'url': url
        })

    def get_dest_dir(self, name):
        url = urlparse(name)
        return os.path.join(self.dest_dir, url.netloc, url.path.removeprefix('/'))

    def error_repo(self, path):
        self.log.warning(f'error update repo: {path}')
        archive_path = os.path.join(self.archive_dir, 'error', datetime.now().strftime('%Y%m%d/%H%M'))
        shutil.move(path, os.path.join(archive_path, path.removeprefix(self.dest_dir).removeprefix('/')))

    def make_mirror(self):
        for repo in self.repos:
            path = self.get_dest_dir(repo.get('url'))

            if path.endswith('.wiki.git') and not self.git_lsremote(repo.get('clone_url')):
                continue

            repo_id = repo.get('id')
            if repo_id and path.endswith('.wiki.git'):
                path = path[:-9] + f'-{repo_id}' + path[-9:]
            elif repo_id and path.endswith('.git'):
                path = path[:-4] + f'-{repo_id}' + path[-4:]

            if not os.path.exists(path):
                os.makedirs(path, exist_ok=True)
                self.log.warning(f'add new repo: {path}')
                try:
                    git.Repo.clone_from(repo.get('clone_url'), path, bare=True, mirror=True)
                except git.exc.InvalidGitRepositoryError:
                    self.error_repo(path)

            else:
                self.log.warning(f'updating repo: {path}')
                try:
                    git_repo = git.Repo(path, search_parent_directories=True)
                    for remote in git_repo.remotes:
                        remote.fetch('+refs/heads/*:refs/remotes/origin/*')
                except git.exc.InvalidGitRepositoryError:
                    self.error_repo(path)

    def get_json(self, url):
        resp = self.session.get(url)
        resp.raise_for_status()
        return resp.json()

    @staticmethod
    def git_lsremote(url):
        g = git.cmd.Git()
        try:
            g.ls_remote(url)
        except git.exc.GitCommandError:
            return None

        return True


class Github(GitBackup):
    def __init__(self, creds=None):
        super(Github, self).__init__()
        self.github_api = os.getenv('GITHUB_API_ADDR', 'api.github.com')
        
        if creds:
            self.session.auth = creds

    @staticmethod
    def _get_wiki_path(path):
        if path.endswith('.git'):
            path = path[:-4] + f'.wiki' + path[-4:]
        else:
            path += '.wiki'
        return path

    def _process_repo(self, repo):
        if os.getenv('GITHUB_WIKI') and repo.get('has_wiki'):
            self.add_repo(
                repo.get('id'),
                self._get_wiki_path(repo.get('ssh_url')),
                self._get_wiki_path(repo.get('clone_url'))
            )

        self.add_repo(repo.get('id'), repo.get('ssh_url'), repo.get('clone_url'))

    def get_repos(self):
        user = self.get_json(f'https://{self.github_api}/user')
        
        resp = self.session.get(f'https://{self.github_api}/user/repos')
        resp.raise_for_status()
        repos = resp.json()

        while 'next' in resp.links.keys():
            resp = self.session.get(resp.links['next']['url'])
            resp.raise_for_status()
            repos.extend(resp.json())

        for repo in repos:
            self._process_repo(repo)
            
        if os.getenv('GITHUB_STARRED'):            
            resp_star = self.session.get(uri_expand(user.get('starred_url')))
            resp_star.raise_for_status()
            starred = resp_star.json()
            
            while 'next' in resp_star.links.keys():
                resp_star = self.session.get(resp_star.links['next']['url'])
                resp_star.raise_for_status()
                starred.extend(resp_star.json())
            
            for repo in starred:
                self._process_repo(repo)


if __name__ == '__main__':
    github_login = os.getenv('GITHUB_LOGIN')
    github_token = os.getenv('GITHUB_TOKEN')

    if github_login and github_token:
        creds = (github_login, github_token)
        Github(creds).run()
    else:
        print('Access is not configured')
