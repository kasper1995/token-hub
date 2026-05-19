---
name: token-hub-production-deploy
description: Use when publishing TokenHub or new-api changes to production, especially production deploy, 发布到生产, new-api.service, or ssh root@182.92.166.143.
---

# TokenHub Production Deploy

## Core Rule

Do not guess production paths or ports. Preserve existing ports, security groups, service files, and proxy config unless explicitly asked to change them. Verify service state, checksum, health endpoint, and logs before claiming success.

## Production Facts

| Item | Value |
|---|---|
| SSH | `root@182.92.166.143` |
| Service | `new-api.service` |
| Binary | `/srv/new-api/current/new-api` |
| Env | `/srv/new-api/current/.env` |
| Backend port | Existing `new-api.service` uses `3202`; do not change |
| Frontend port | Existing frontend access commonly uses `3200`; do not change |
| Health check | `curl -fsS http://127.0.0.1:3202/api/status` on the server |
| Runtime | systemd + MySQL, not Docker |

## Port Discipline

Security groups are preconfigured. Binary deploy only replaces the backend binary and restarts `new-api.service`; never edit `--port`, frontend port `3200`, Nginx/proxy, firewall, cloud security groups, or Docker/network settings. If a command exposes a new port, stop and ask.

## Preconditions

1. Show deploy candidate: `git status --short` and changed files.
2. Run targeted local verification.
3. Build embedded frontends first:
   - `npm --prefix web/default run build`
   - `npm --prefix web/classic run build`
4. Build Linux x86_64 backend locally:
   - `VERSION_VALUE=$(< VERSION); go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$VERSION_VALUE'" -o /tmp/token-hub-new-api .`

## Deploy Commands

```bash
sha256sum /tmp/token-hub-new-api
scp /tmp/token-hub-new-api root@182.92.166.143:/tmp/token-hub-new-api.deploy
ssh root@182.92.166.143 'set -e; \
  sha256sum /tmp/token-hub-new-api.deploy; \
  file /tmp/token-hub-new-api.deploy; \
  ldd /tmp/token-hub-new-api.deploy | sed -n "1,80p"; \
  ts=$(date +%Y%m%d%H%M%S); \
  cp -p /srv/new-api/current/new-api /srv/new-api/current/new-api.bak.$ts; \
  install -o root -g root -m 0755 /tmp/token-hub-new-api.deploy /srv/new-api/current/new-api; \
  systemctl restart new-api; \
  systemctl is-active new-api; \
  systemctl status new-api --no-pager | sed -n "1,25p"'
```

## Post-Deploy Verification

```bash
ssh root@182.92.166.143 'set -e; \
  systemctl is-active new-api; \
  ps -o pid,lstart,etime,args -p $(systemctl show -p MainPID --value new-api); \
  curl -fsS http://127.0.0.1:3202/api/status | head -c 1200; \
  sha256sum /srv/new-api/current/new-api'

ssh root@182.92.166.143 'journalctl -u new-api --since "2 minutes ago" --no-pager | grep -Ei "panic|fatal|error|failed" || true'
```

For UI changes, check `/api/status` `theme`. Production currently serves `classic`; default-only pages will not appear in the active UI. New routes: verify explicitly. Protected admin APIs should return `401`/`403` unauthenticated, not `404`.

## Remote Build Caveats

Do not rely on production builds until revalidated: Node was `18.19.1` vs frontend `>=20.19`; temporary Node `24.10.0` hit `SIGBUS` in `rsbuild build`. Go was `1.22.2` vs repo `go 1.25.1`. Safe path: build frontends and backend locally; Go embeds `web/default/dist` and `web/classic/dist`.

## Rollback

```bash
ssh root@182.92.166.143 'ls -lt /srv/new-api/current/new-api.bak.* | head; \
  install -o root -g root -m 0755 /srv/new-api/current/<backup-file> /srv/new-api/current/new-api; \
  systemctl restart new-api; systemctl is-active new-api'
```

Use the newest known-good backup.

## Common Mistakes

| Mistake | Correct action |
|---|---|
| Guessing path/ports | Query `systemctl status new-api`; preserve existing ports. |
| Changing network exposure | Do not edit security group/firewall/proxy during binary deploy. |
| Building Go before frontend | Build frontend first; Go embeds both `dist` dirs. |
| Claiming success after restart | Verify active service, health, checksum, and logs. |
| Forgetting backup | Copy old binary to `new-api.bak.<timestamp>` first. |
