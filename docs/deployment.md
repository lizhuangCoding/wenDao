# Production Deployment

This project can be deployed as one Docker Compose stack:

- Caddy terminates HTTPS and routes traffic.
- The frontend is built by Vite and served by Nginx.
- The backend runs the Go API on port `8089`.
- MySQL stores application data.
- Redis Stack is used for Redis plus vector search.
- Uploaded files are stored in the `uploads_data` Docker volume.

## 1. Prepare the server

Install Docker Engine and the Docker Compose plugin on your server. Point your domain's A record to the server IP before starting Caddy, otherwise HTTPS certificate issuance can fail.

If the server is in mainland China and the domain resolves to it, complete ICP filing first.

## 2. Configure secrets

Copy the example environment file and edit the values:

```bash
cp .env.production.example .env.production
```

Required values:

- `PUBLIC_DOMAIN`
- `SITE_URL`
- `DB_PASSWORD`
- `MYSQL_ROOT_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`

Optional values:

- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `GITHUB_CALLBACK_URL`
- `DOUBAO_API_KEY`
- `DOUBAO_ENDPOINT`
- `DOUBAO_CHAT_MODEL`
- `DOUBAO_EMBEDDING_MODEL`
- `RESEARCH_ENDPOINT`
- `RESEARCH_API_KEY`

## 3. Build and start

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Check service status:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f backend
```

### IP-only deployment

When deploying by public IP before a domain is ready, use the IP override file. It disables Caddy, exposes the frontend on server port `8081`, and uses the images that are easier to pull on Alibaba Cloud Linux:

For IP-only deployment, set the public URL values in `.env.production` with the port included:

```env
PUBLIC_DOMAIN=http://your-server-ip:8081
SITE_URL=http://your-server-ip:8081
GITHUB_CALLBACK_URL=http://your-server-ip:8081/api/auth/github/callback
```

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml up -d --build
```

Check service status:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml logs --tail=100 backend
```

Then visit:

```text
http://your-server-ip:8081
```

### GitHub Actions SSH deployment

The repository includes `.github/workflows/deploy.yml` for automatic deployment from the `main` branch. The workflow SSHes into the server, pulls the latest code, then rebuilds and restarts the Docker Compose stack on the server.

Create these GitHub repository secrets:

- `DEPLOY_HOST`: server IP or hostname, for example `1.2.3.4`
- `DEPLOY_USER`: SSH user, for example `root`
- `DEPLOY_SSH_KEY`: private key whose public key is in the server user's `~/.ssh/authorized_keys`
- `DEPLOY_PATH`: absolute path to the checked-out repository on the server, for example `/root/wenDao`
- `DEPLOY_PORT`: optional SSH port; defaults to `22` when omitted

For IP-only deployment, no repository variables are required. The workflow uses these compose files by default:

```text
-f docker-compose.prod.yml -f docker-compose.ip.yml
```

For domain deployment with Caddy enabled, set repository variable `DEPLOY_COMPOSE_FILES` to:

```text
-f docker-compose.prod.yml
```

If the frontend needs a custom build-time API base URL, set `VITE_API_BASE_URL` in the server-side `.env.production`. The Compose build arg defaults to `/api`.

The frontend Docker build defaults `FRONTEND_NODE_OPTIONS` to `--max-old-space-size=1536` so Vite has enough Node heap during `npm run build` without pushing small servers as hard as a 2G heap. If your server has very little memory, keep the swap setup below; if the build still hits a V8 heap limit, raise `FRONTEND_NODE_OPTIONS` in `.env.production`.

Because this deployment builds frontend and backend images on the server, make sure the server has enough memory or swap. On small servers, a 4G swap file is usually enough to avoid `npm run build` being killed:

```bash
fallocate -l 4G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
grep -q '^/swapfile ' /etc/fstab || echo '/swapfile none swap sw 0 0' >> /etc/fstab
free -h
```

Prepare the server once:

```bash
cd /root/wenDao
git fetch origin main
git checkout -B main origin/main
test -f .env.production
docker compose version
```

The server checkout must be able to pull from GitHub without interactive input. For a private repository, add a deploy key to the GitHub repository and install the matching private key on the server, or configure HTTPS credentials for the server's git remote. Keep `.env.production` only on the server; do not commit it.

After secrets are configured, push to `main` or run the workflow manually from GitHub Actions. The deployment step runs the equivalent of:

```bash
git fetch --prune origin main
git checkout -B main origin/main
git reset --hard origin/main
docker compose --env-file .env.production \
  -f docker-compose.prod.yml \
  -f docker-compose.ip.yml \
  up -d --build
```

Do not keep uncommitted tracked changes on the server checkout. The workflow resets tracked files to `origin/main` before restarting the stack. Ignored runtime files such as `.env.production` are preserved.

## 4. Create the first admin user

Run the admin bootstrap command inside the backend container. It creates the user if it does not exist, or promotes an existing user with the same email to `admin`.

If `ADMIN_EMAIL`, `ADMIN_USERNAME`, and `ADMIN_PASSWORD` are already set in `.env.production`, run:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml exec \
  backend /app/wendao-init-admin
```

Or pass the values only for this command:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml exec \
  -e ADMIN_EMAIL="your-email@example.com" \
  -e ADMIN_USERNAME="your-admin-username" \
  -e ADMIN_PASSWORD="replace-with-a-strong-password" \
  backend /app/wendao-init-admin
```

Do not commit real admin passwords to Git. Pass them only at runtime.

## 5. Backups

Back up at least these volumes:

- `mysql_data`
- `redis_data`
- `uploads_data`

For a quick MySQL dump:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml exec mysql \
  mysqldump -uroot -p wendao > wendao.sql
```
