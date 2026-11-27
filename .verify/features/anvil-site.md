# Feature: anvil-site

Product landing page for Anvil at `site/` — Next.js + Tailwind, YC-style
(Resend/Linear/Anode aesthetic): obsidian dark surfaces, terminal hero
simulation, spotlight bento cards, smooth scroll. Served at
`anvil.abhiudayg.dev` via Vercel + Cloudflare DNS; GH Actions deploys on main.

## Sub-features

- `hero-terminal`: hero headline + auto-typing terminal cycling the stage
  loop (`anvil start` → `build` → question → `grade` → PERFECT → `unlock` →
  fix → `check` ✓) with a copyable install command.
- `pipeline`: the evaluation loop rendered as numbered stages.
- `drift-compare`: split panel — tutorial-watching status quo vs Anvil's
  guarded-build loop.
- `pillars-bento`: asymmetric spotlight cards — guard hook, real git
  workspaces, deterministic CLI grading, tutor subagent, TUI.
- `quickstart`: numbered copyable command cards.
- `obsidian-theme`: `#08090b` base + 32px grid mesh, `#0c0e14` panels,
  `white/[0.08]` borders, radial blooms, mono accents.
- `deploy-CI`: `.github/workflows/site.yml` builds and deploys `site/` to
  Vercel production on pushes to main touching the site.

## How to get to it (user POV)

- `pnpm -C site dev` → localhost preview.
- `pnpm -C site build` → static/SSG production build.
- `https://anvil.abhiudayg.dev` → live page after DNS + Vercel wiring.

## Verification Suites (Targeted Test Commands)

- Unit: `pnpm -C site lint` (eslint zero errors).
- Functional: `pnpm -C site build` then grep SSG HTML for each beat's text.
- Allowlisted: `sh .verify/run.sh anvil-site`

1. Request: `pnpm -C site build`.
   Why: the page must compile and prerender without remote font fetches.
   Expected: exit 0; `.next/` output exists.
   Impact: production bundle produced.
2. Request: grep prerendered `index.html` for `anvil start`, `PERFECT`,
   `unlock`, `check`.
   Why: the hero terminal must carry the real stage loop, not lorem.
   Expected: all four strings present.
   Impact: none.
3. Request: grep for `Teach`, `Build`, `Grade`, `Unlock`, `Check`, `Review`.
   Why: the pipeline beat must mirror the evaluation loop in AGENTS.md.
   Expected: all six stage names present.
   Impact: none.
4. Request: grep for guard/workspace/tutor/TUI pillar keywords.
   Why: bento cards must name the real mechanisms.
   Expected: `guard`, `workspace`, `tutor` present.
   Impact: none.
5. Request: `cf dns records list -z abhiudayg.dev | grep anvil`.
   Why: the subdomain must resolve to Vercel.
   Expected: CNAME `anvil` → `cname.vercel-dns.com` (DNS only, not proxied).
   Impact: none on zone state beyond the record itself.
6. Request: HTTP GET `https://anvil.abhiudayg.dev`.
   Why: end-to-end proof the deploy + DNS + TLS chain works.
   Expected: 200 and body contains `Anvil`.
   Impact: none.

## Gotchas

- `next build` must not fetch Google fonts — system font stack only
  (sandbox/offline builds).
- Vercel external-DNS verification wants the CNAME **DNS-only** (grey cloud),
  not proxied, or verification/SSL issuance can loop.
- `.vercel/` holds orgId/projectId — gitignored; CI re-derives linkage via
  `VERCEL_ORG_ID`/`VERCEL_PROJECT_ID` secrets.
- `VERCEL_TOKEN` is set from the local CLI auth (device-login token) — if CI
  401s, rotate it at vercel.com/account/tokens.
- `vercel project create` defaults framework to `Other` → the deploy serves
  NOT_FOUND on every route until `framework: "nextjs"` is PATCHed via the API.
- Vercel project `nodeVersion` must be ≤ 24.x — build-utils rejects 26.x.
- `ssoProtection` is `all_except_custom_domains`: `*.vercel.app` URLs show a
  login wall, the custom domain stays public.
- `pnpm/action-setup` needs an explicit `version:` — the repo root is a Go
  module with no package.json `packageManager` field.
- No `h-full`/`overflow-x-hidden` on `html`/`body` (Lenis frozen-scroll trap).
