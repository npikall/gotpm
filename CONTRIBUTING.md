# Contributing

Contributions are welcome, and they are greatly appreciated! Every little bit helps, and credit will always be given.

You can contribute in many ways:

- Report Bugs
- Fix Bugs
- Implement Features
- Write Documentation
- Submit Feedback

## Get Started

Ready to contribute? Here's how to set up `gotpm` for local development.

- Fork the `gotpm` repo on GitHub.

- Clone your fork locally:

    ```sh
    git clone git@github.com:<YOUR_GH_USER>/gotpm.git
    ```

- Create a branch for local development:

    ```sh
    git checkout -b name-of-your-bugfix-or-feature
    ```

    Now you can make your changes locally.

- When you're done making changes, check that your changes are formatted
  correctly and the tests are passing.

    > [!TIP]
    > Install the [`go-task`][go-task] Taskrunner.

    ```sh
    # format the code
    task format

    # run the tests
    task test
    ```

- Commit your changes and push your branch to GitHub.

- Submit a pull request through the GitHub website.

## Documentation

The site is built with [zensical][zensical] from `zensical.toml` and the
Markdown in `docs/`. Install it with the `markdown-callouts` extension, which
is what renders GitHub alerts (`> [!TIP]`) as admonitions on the site:

```sh
# once
uv tool install zensical --with markdown-callouts
```

```sh
# serve the site locally at http://localhost:8000
task docs:serve

# build it into site/
task docs:build
```

The command reference embeds the binary's own help text, generated into
`docs/includes/cli/`. Those files are committed, and CI fails when they are
stale, so regenerate them after changing a command's help:

```sh
task docs:cli
```

Prose follows the vocabulary in [`CONTEXT.md`](https://github.com/npikall/gotpm/blob/main/CONTEXT.md) - it is the glossary
the docs and the CLI strings are both written against. The decision records in
[`docs/adr/`](https://github.com/npikall/gotpm/tree/main/docs/adr) explain why the
non-obvious parts are the way they are; they are for people working on gotpm and
are deliberately not published to the site.

[go-task]: <https://github.com/go-task/task>
[zensical]: <https://zensical.org>
