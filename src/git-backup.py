import logging
import os.path
import requests
import shutil
import git

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
        shutil.rmtree(path)

    def make_mirror(self):
        for repo in self.repos:
            path = self.get_dest_dir(repo.get('url'))

            repo_id = repo.get('id')
            if repo_id and path.endswith('.git'):
                path = path[:-4] + f'-{ids}' + path[-4:]

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


class Github(GitBackup):
    def __init__(self, creds=None):
        super(Github, self).__init__()
        self.github_api = os.getenv('GITHUB_API_ADDR', 'api.github.com')
        
        if creds:
            self.session.auth = creds

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
            self.add_repo(repo.get('id'), repo.get('ssh_url'), repo.get('clone_url'))
            
        if os.getenv('GITHUB_STARRED'):            
            resp_star = self.session.get(uri_expand(user.get('starred_url')))
            resp_star.raise_for_status()
            starred = resp_star.json()
            
            while 'next' in resp_star.links.keys():
                resp_star = self.session.get(resp_star.links['next']['url'])
                resp_star.raise_for_status()
                starred.extend(resp_star.json())
            
            for repo in starred:
                self.add_repo(repo.get('id'), repo.get('ssh_url'), repo.get('clone_url'))                


if __name__ == '__main__':
    github_login = os.getenv('GITHUB_LOGIN')
    github_token = os.getenv('GITHUB_TOKEN')

    if github_login and github_token:
        creds = (github_login, github_token)
        Github(creds).run()
    else:
        print('Access is not configured')
