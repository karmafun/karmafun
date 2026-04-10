from typing import cast

import re
from mkdocs.config.defaults import MkDocsConfig
from mkdocs.structure.pages import Page
from mkdocs.structure.files import Files
from mkdocs.utils import normalize_url


GIT_LATEST_RELEASE = 'v0.1.0'  # Placeholder for the latest release tag
GIT_VERSION_PATTERN = re.compile(r'^v\d+\.\d+\.\d+$')  # Pattern to match version tags like v1.2.3

# There are three ways to get the latest release tag:
# 1. Use the GitHub API to fetch the latest release tag.
# 2. Use the `git` command to get the latest release tag from the local repository.
# 3. Use the GITHUB_REF environment variable, which is set by GitHub Actions when the workflow is triggered by a
#    release event.


def get_latest_release_from_github_api(repo_name: str) -> str:
    import requests

    response = requests.get(f'https://api.github.com/repos/{repo_name}/releases/latest')
    response.raise_for_status()
    return response.json()['tag_name']

def get_latest_release_from_git_command() -> str:
    import subprocess

    result = subprocess.run(['git', 'describe', '--abbrev=0', '--tags', '--match=v[0-9]*'], capture_output=True, text=True)
    result.check_returncode()
    return result.stdout.strip()

def get_latest_release_from_github_ref() -> str:
    import os

    github_ref = os.getenv('GITHUB_REF', '')
    if github_ref.startswith('refs/tags/'):
        tag = github_ref[len('refs/tags/'):]
        if GIT_VERSION_PATTERN.match(tag):
            return tag
    raise ValueError('GITHUB_REF does not contain a valid tag reference')

def on_config(config: MkDocsConfig):
    global GIT_LATEST_RELEASE
    repo_name = config.repo_name
    try:
        GIT_LATEST_RELEASE = get_latest_release_from_github_api(repo_name)
    except Exception as e:
        print(f'Error fetching latest release from GitHub API: {e}')
        try:
            GIT_LATEST_RELEASE = get_latest_release_from_git_command()
        except Exception as e:
            print(f'Error fetching latest release from git command: {e}')
            try:
                GIT_LATEST_RELEASE = get_latest_release_from_github_ref()
            except Exception as e:
                print(f'Error fetching latest release from GITHUB_REF: {e}')
            # Fallback to a default value if all methods fail
            GIT_LATEST_RELEASE = 'v0.1.0'

def on_page_markdown(output: str, *, page: Page, config: MkDocsConfig, files: Files) -> str:
    if '{git_latest_release}' in output:
        return output.replace('{git_latest_release}', GIT_LATEST_RELEASE)
    return output
