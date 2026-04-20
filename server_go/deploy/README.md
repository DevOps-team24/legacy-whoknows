# Deploy – WhoKnows Go

Appen kører på en Azure VM (`20.240.47.24` / `huw.dk`) som Docker-container bag Nginx med HTTPS.

Alle i gruppen deployer som den fælles bruger **deploy** via SSH-nøgler. Der er ingen fælles adgangskode.

---

## Oversigt

| Komponent    | Beskrivelse                                           |
| ------------ | ----------------------------------------------------- |
| **Go-app**   | Lytter på `127.0.0.1:8080` (kun internt)              |
| **Docker**   | Bygger og kører Go-appen via `docker compose`         |
| **Nginx**    | Reverse proxy på port 80/443, HTTPS via Let's Encrypt |
| **Database** | PostgreSQL 17, data i volume `/opt/whoknows/pgdata`   |

### Filstruktur

```text
deploy/
├── Dockerfile                  # Go-image (multi-stage build)
├── docker-compose.server.yml   # Compose-fil brugt på serveren
├── env.example                 # Reference for env-vars (den rigtige .env skrives af deploy-workflow fra GitHub Secrets)
├── README.md                   # Denne fil
└── ansible/
    ├── ansible.cfg
    ├── inventory.yml
    ├── playbook.yml
    └── roles/
        ├── docker/             # Installerer Docker + Compose
        ├── whoknows-app/       # Opretter dirs, kopierer kilde, bygger og starter
        └── nginx/              # Installerer Nginx, Certbot og HTTPS
```

---

## Første gang – serveropsætning

Disse trin er allerede udført på `20.240.47.24`. De står her som reference, hvis serveren skal sættes op igen.

### 1. Manuelle trin (før Ansible)

**Azure-porte** – åbn i Azure Portal → VM → Networking:

- **Port 22** (SSH), **80** (HTTP), **443** (HTTPS): Allow (TCP).

**DNS** – peg domænet (`huw.dk`) til serverens IP via en A-record.

**Opret deploy-bruger** – SSH ind som en bruger med sudo (fx `adminuser`):

```bash
sudo adduser --disabled-password --gecos "" deploy
sudo usermod -aG sudo deploy
sudo mkdir -p /home/deploy/.ssh
sudo cp /home/adminuser/.ssh/authorized_keys /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh
sudo chmod 600 /home/deploy/.ssh/authorized_keys
```

### 2. Kør Ansible

Ansible klarer resten: Docker, app-deploy, Nginx og Certbot/HTTPS.

```bash
cd path/til/legacy-whoknows/server_go
ansible-playbook deploy/ansible/playbook.yml
```

### 3. Importer eksisterende SQLite-data (valgfrit)

Hvis I har en legacy `whoknows.db`-fil I vil migrere:

```bash
# Fra din egen maskine med Postgres DSN mod produktion (via SSH-tunnel e.l.)
export DATABASE_URL="postgres://user:pass@host:5432/whoknows?sslmode=disable"
go run ./cmd/import-sqlite -sqlite path/to/whoknows.db
```

Goose kører automatisk på server-start, så skemaet er allerede på plads når import køres.

### Resultat

- **<https://huw.dk/>** – virker med grøn lås.
- **<http://huw.dk/>** – omdirigerer til **<https://huw.dk/>**.
- **<http://20.240.47.24>** – omdirigerer til **<https://huw.dk>**.
- **<https://20.240.47.24>** – omdirigerer til **<https://huw.dk>**.

---

## Deploy (fremover)

Alt deploy sker via Ansible. Én kommando installerer Docker, deployer appen, sætter Nginx op og henter SSL-certifikat.

### Forudsætninger på din PC

1. **Ansible** installeret (kræver Linux, Mac eller WSL). Installer med `pip install ansible`.
2. **SSH-nøgle** – din public key skal ligge i `deploy`-brugerens `~/.ssh/authorized_keys` på serveren.

### Kør deploy

Kør altid fra `server_go`-mappen:

```bash
cd path/til/legacy-whoknows/server_go
ansible-playbook deploy/ansible/playbook.yml
```

### Hvad Ansible gør

Playbooken (`deploy/ansible/playbook.yml`) kører tre roller:

1. **docker** – Installerer Docker + Compose og tilføjer `deploy` til docker-gruppen.
2. **whoknows-app** – Opretter mapper, kopierer kildekode, templater `docker-compose.yml` og `.env`, bygger image og starter containeren.
3. **nginx** – Installerer Nginx og Certbot, placerer site-config med HTTPS og IP-redirect, henter SSL-certifikat fra Let's Encrypt.

Postgres-data ligger i volumen `/opt/whoknows/pgdata/` og bevares mellem deploys. `goose` kører automatisk ved app-start og anvender alle nye migrations fra `server_go/migrations/`.

### Konfiguration

Rediger `deploy/ansible/inventory.yml`:

- `ansible_host`: serverens IP.
- `ansible_user`: fælles bruger (fx `deploy`).
- `domain`: domænenavn (fx `huw.dk`).
- `certbot_email`: email til Let's Encrypt-advarsler.

### Nyttige kommandoer

```bash
# Test SSH-forbindelse
ansible all -m ping

# Kun Docker-installation
ansible-playbook deploy/ansible/playbook.yml --tags docker

# Kun app-deploy
ansible-playbook deploy/ansible/playbook.yml --tags app

# Kun Nginx + Certbot
ansible-playbook deploy/ansible/playbook.yml --tags nginx

# Tørkørsel (ingen ændringer)
ansible-playbook deploy/ansible/playbook.yml --check
```

---

## Database (PostgreSQL + goose)

### Credentials

Ingen creds ligger i repoet. `.env` på serveren skrives én gang af Ansible via `env.j2`, som bruger env-var-lookups:

- `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` – eksporteres som env-vars før `ansible-playbook` køres.
- `DATABASE_URL` genereres automatisk i `env.j2` ud fra ovenstående.

### Migrations

Migrations ligger i `server_go/migrations/` i goose-format. Ved server-start kører appen `goose.Up(...)` mod den aktive database. For at tilføje en ny migration lokalt:

```bash
cd server_go
goose -dir migrations create add_noget_nyt sql
# rediger filen, commit + push — CI validerer den, deploy kører den automatisk
```

### Lokal dev-DB

```bash
cd server_go
cp .env.example .env   # rediger med dine lokale values
docker compose -f docker-compose.dev.yml up -d
```

### Backup på produktion

```bash
ssh deploy@<server-ip> "docker exec whoknows-postgres pg_dump -U <user> <db>" > whoknows-$(date +%F).sql
```

### Tjek logs

```bash
ssh deploy@<server-ip> "cd /opt/whoknows && docker compose logs -f whoknows-blue whoknows-green postgres"
```

---

## På serveren (nyttige kommandoer)

```bash
ssh deploy@20.240.47.24
cd /opt/whoknows
docker compose ps              # Status
docker compose logs -f whoknows # Følg logs
docker compose restart whoknows # Genstart
docker compose down             # Stop alt
docker compose up -d            # Start alt
```
