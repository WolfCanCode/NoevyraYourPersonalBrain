# Publish on WolfCanCode GitHub

The starter Git repository already has the local commit author:

```text
WolfCanCode <wolfcancode@gmail.com>
```

**One-time step:** Create a new **public empty** GitHub repository named `personal-ai-brain` under the `WolfCanCode` account. Do not initialize the repository with README / .gitignore / license — these files already exist locally.

Then from the repository directory:

```bash
git remote -v
git log -1 --format='%an <%ae>'
git push -u origin main
```

If GitHub Pages is desired, open repo **Settings → Pages**, choose GitHub Actions as its source. A deploy workflow is included in `.github/workflows/pages.yml`.

Security note: `wolfcancode@gmail.com` will be visible as the Git author email in a public commit. If you prefer to hide an email publicly, use a GitHub no-reply commit address instead.

## Vercel landing page

Deploy the static marketing site using the checked-in `vercel.json` (`outputDirectory: site`).
See [docs/VERCEL.md](docs/VERCEL.md) for Git-linked and CLI deployment instructions.
This deploys only the public static page, **not** personal Brain memory or the CLI backend.
