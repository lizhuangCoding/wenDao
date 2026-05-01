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

## 4. Create the first admin user

Run the admin bootstrap command inside the backend container. It creates the user if it does not exist, or promotes an existing user with the same email to `admin`.

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml exec \
  -e ADMIN_EMAIL="your-email@example.com" \
  -e ADMIN_USERNAME="your-admin-username" \
  -e ADMIN_PASSWORD="replace-with-a-strong-password" \
  backend /app/wendao-init-admin
```

For multiple admins, use numbered variables:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml exec \
  -e ADMIN_EMAIL_1="first@example.com" \
  -e ADMIN_USERNAME_1="first-admin" \
  -e ADMIN_PASSWORD_1="replace-with-a-strong-password" \
  -e ADMIN_EMAIL_2="second@example.com" \
  -e ADMIN_USERNAME_2="second-admin" \
  -e ADMIN_PASSWORD_2="replace-with-another-strong-password" \
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
