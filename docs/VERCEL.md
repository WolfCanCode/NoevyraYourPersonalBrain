# Deploy the landing page on Vercel

Personal AI Brain is a **local-first CLI**, not a cloud backend. This Vercel deployment publishes the **static landing page only** from `site/`.

## Option A: Deploy from a GitHub repository (recommended)

1. Create `WolfCanCode/personal-ai-brain` as a public GitHub repository (if not already present).
2. Push the source repository to GitHub.
3. In Vercel, import the GitHub repository.
4. Choose the **Other** framework preset and keep the **Root Directory** at the repository root.
5. The committed `vercel.json` sets `outputDirectory` to `site`, so only the landing page is published.
6. Deploy. Once Git is linked, future pushes to the production branch redeploy automatically.

## Option B: Deploy directly from a local working copy

Prerequisite: a Vercel account and a Vercel CLI session authenticated with `vercel login`.

```sh
npx vercel link
npx vercel deploy
# Verify the preview, then:
npx vercel deploy --prod
```

Run those commands from the **repository root**. The Vercel CLI prints the actual deployment URL.

## Notes

- Do not deploy the local SQLite brain database, private memory, secrets, or agent configuration.
- Do not add production secrets to `vercel.json` or GitHub.
- The Vercel site and the GitHub repository are public, but Brain data stays local.
- The static site does not run daily sync. That runs on the user's device where the CLI is installed.
- The landing page's GitHub button will work after `WolfCanCode/personal-ai-brain` exists.
