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

When deploying by public IP before a domain is ready, use the IP override file. It disables Caddy, exposes the frontend directly on port `80`, and uses the images that are easier to pull on Alibaba Cloud Linux:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml up -d --build
```

Check service status:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml ps
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml logs --tail=100 backend
```

## 4. Create the first admin user

Register the first user from the site, then promote that user in MySQL. Adjust the username or email condition to match your account:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml exec mysql \
  mysql -uwendao -p wendao \
  -e "UPDATE users SET role = 'admin' WHERE username = 'your-username';"
```

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
