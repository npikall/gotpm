# /// script
# requires-python = ">=3.14"
# dependencies = [
#     "httpx>=0.28.1",
# ]
# ///
import asyncio
import re
from pathlib import Path

import httpx

type LatestVersions = dict[str, str | None]

ACTION_REF_PATTERN = re.compile(
    r"(uses:\s+)([a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+)@(v\d[\w.]*)"
)


def main() -> None:
    github_dir = get_project_root() / ".github"
    repos = scan_action_repos(github_dir)
    print(f"found {len(repos)} unique action repos: {repos}")
    latest_versions = asyncio.run(fetch_latest_versions(repos))
    apply_updates(github_dir, latest_versions)
    print("done.")


def scan_action_repos(github_dir: Path) -> list[str]:
    repos: set[str] = set()
    for workflow in github_dir.rglob("*.yml"):
        for m in ACTION_REF_PATTERN.finditer(workflow.read_text()):
            repos.add(m.group(2))
    return sorted(repos)


def apply_updates(
    github_dir: Path,
    latest_versions: LatestVersions,
) -> None:
    # TODO: update all files asyncronosly, if it makes sense
    for workflow in github_dir.rglob("*.yml"):
        _update_workflow_file(workflow, latest_versions)


def _update_workflow_file(
    file: Path,
    latest_versions: LatestVersions,
) -> None:
    original = file.read_text()

    # TODO: extract function, and make function name more expressive
    def replace(m: re.Match[str]) -> str:
        prefix, repo, current_pin = m.group(1), m.group(2), m.group(3)
        latest_tag = latest_versions.get(repo)
        if latest_tag is None:
            return m.group(0)
        if latest_tag == current_pin:
            return m.group(0)
        print(f"{file.name}: {repo}@{current_pin} -> @{latest_tag}")
        return f"{prefix}{repo}@{latest_tag}"

    updated = ACTION_REF_PATTERN.sub(replace, original)
    if updated != original:
        file.write_text(updated)


async def fetch_latest_versions(
    repos: list[str],
) -> LatestVersions:
    """Fetch the latest release tag for every repo in a single HTTP session."""
    async with httpx.AsyncClient() as client:
        tags = await asyncio.gather(
            *[_fetch_latest_github_tag(repo, client) for repo in repos]
        )
    return dict(zip(repos, tags))


async def fetch_latest(repo: str, client: httpx.AsyncClient) -> tuple[str, str | None]:
    latest_tag = await _fetch_latest_github_tag(repo, client)
    return repo, latest_tag


async def _fetch_latest_github_tag(repo: str, client: httpx.AsyncClient) -> str | None:
    url = f"https://api.github.com/repos/{repo}/releases/latest"
    resp = await client.get(url, timeout=10)
    if resp.status_code != httpx.codes.OK:
        print(f"GitHub fetch failed for {repo}: HTTP {resp.status_code}")
        return None
    return resp.json().get("tag_name")


def get_project_root(marker: str = "go.mod") -> Path:
    for parent in Path(__file__).parents:
        if (parent / marker).exists():
            return parent
    raise FileNotFoundError(f"no '{marker}' found in any parent directory")


if __name__ == "__main__":
    main()
