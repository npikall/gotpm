# /// script
# requires-python = ">=3.14"
# dependencies = [
#     "httpx>=0.28.1",
#     "rich>=15.0.0",
# ]
# ///
import asyncio
import re
from pathlib import Path

import httpx  # ty:ignore[unresolved-import]
from rich import print  # ty:ignore[unresolved-import]

type LatestVersions = dict[str, str | None]

ACTION_REF_PATTERN = re.compile(
    r"(uses:\s+)([a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+)@(v\d[\w.]*)"
)


async def main() -> None:
    github_dir = get_project_root() / ".github"
    repos = scan_action_repos(github_dir)
    print(f"found {len(repos)} unique action repos: {repos}")
    latest_versions = await fetch_latest_versions(repos)
    await apply_updates(github_dir, latest_versions)
    print("done")


def scan_action_repos(github_dir: Path) -> list[str]:
    repos: set[str] = set()
    for workflow in github_dir.rglob("*.yml"):
        for m in ACTION_REF_PATTERN.finditer(workflow.read_text()):
            repos.add(m.group(2))
    return sorted(repos)


async def apply_updates(
    github_dir: Path,
    latest_versions: LatestVersions,
) -> None:
    workflows = list(github_dir.rglob("*.yml"))
    await asyncio.gather(
        *[_update_workflow_file(wf, latest_versions) for wf in workflows]
    )


def _pin_updated_ref(
    m: re.Match[str],
    file: Path,
    latest_versions: LatestVersions,
) -> str:
    prefix, repo, current_pin = m.group(1), m.group(2), m.group(3)
    latest_tag = latest_versions.get(repo)
    if latest_tag is None or latest_tag == current_pin:
        return m.group(0)
    latest_major = latest_tag.split(".", maxsplit=1)[0]
    print(f"{file.name}: {repo}@{current_pin} -> @{latest_major}")
    return f"{prefix}{repo}@{latest_major}"


async def _update_workflow_file(
    file: Path,
    latest_versions: LatestVersions,
) -> None:
    original = await asyncio.to_thread(file.read_text)
    updated = ACTION_REF_PATTERN.sub(
        lambda m: _pin_updated_ref(m, file, latest_versions), original
    )
    if updated != original:
        await asyncio.to_thread(file.write_text, updated)


async def fetch_latest_versions(
    repos: list[str],
) -> LatestVersions:
    """Fetch the latest release tag for every repo in a single HTTP session."""
    async with httpx.AsyncClient() as client:
        tags = await asyncio.gather(
            *[_fetch_latest_github_tag(repo, client) for repo in repos]
        )
    return dict(zip(repos, tags))


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
    asyncio.run(main())
