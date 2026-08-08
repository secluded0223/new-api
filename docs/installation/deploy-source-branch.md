# Deploy A Source Branch

This guide deploys the current repository branch with Docker Compose. It keeps the existing PostgreSQL and Redis volumes and builds the `new-api` image from source.

## First Deployment

Install Docker Compose on the server, then run:

```bash
sudo mkdir -p /home/ubuntu/new-api
sudo chown "$USER":"$USER" /home/ubuntu/new-api
cd /home/ubuntu/new-api
git clone --branch dev https://github.com/secluded0223/new-api.git .
```

Review `docker-compose.yml` before starting. Change the PostgreSQL and Redis passwords and keep the same values in `SQL_DSN` and `REDIS_CONN_STRING`. Do not expose ports `5432` or `6379` to the public network.

Start the source-built deployment:

```bash
cd /home/ubuntu/new-api
docker compose -f docker-compose.yml -f docker-compose.source.yml up -d --build
docker compose -f docker-compose.yml -f docker-compose.source.yml ps
```

## Update An Existing Deployment

Back up the database and application data before upgrading:

```bash
cd /home/ubuntu/new-api
mkdir -p backups
docker exec postgres pg_dump -U root -d new-api | gzip > "backups/new-api-$(date +%Y%m%d-%H%M%S).sql.gz"
tar -czf "backups/data-$(date +%Y%m%d-%H%M%S).tar.gz" data
```

Pull the branch and rebuild only the application image:

```bash
cd /home/ubuntu/new-api
git fetch origin
git checkout dev
git pull --ff-only origin dev
docker compose -f docker-compose.yml -f docker-compose.source.yml up -d --build new-api
docker compose -f docker-compose.yml -f docker-compose.source.yml ps
docker logs --tail=100 new-api
```

The database and Redis containers are not recreated by the update command, and their named volumes remain unchanged.

## HTTPS

Put Nginx or Caddy in front of the application and proxy HTTPS traffic to `127.0.0.1:3000`. Set these variables in the application service for a public HTTPS deployment:

```yaml
SESSION_SECRET: replace-with-a-long-random-secret
SESSION_COOKIE_SECURE: "true"
SESSION_COOKIE_TRUSTED_URL: https://your-domain.example
```

Keep `SESSION_SECRET` unchanged during normal upgrades. Changing it logs out existing sessions.
