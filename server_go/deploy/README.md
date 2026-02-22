# Deploy – WhoKnows Go

Appen kører på en Azure VM (`20.240.47.24` / `huw.dk`) som Docker-container bag Nginx med HTTPS.

Alle i gruppen deployer som den fælles bruger **deploy** via SSH-nøgler. Der er ingen fælles adgangskode.

---

## Oversigt

| Komponent | Beskrivelse |
|-----------|-------------|
| **Go-app** | Lytter på `127.0.0.1:8080` (kun internt) |
| **Docker** | Bygger og kører Go-appen via `docker compose` |
| **Nginx** | Reverse proxy på port 80/443, HTTPS via Let's Encrypt |
| **Database** | SQLite-fil i `/opt/whoknows/data/whoknows.db` |

### Filstruktur

```
deploy/
├── Dockerfile                  # Go-image (multi-stage build)
├── docker-compose.server.yml   # Compose-fil brugt på serveren
├── env.server                  # Env-vars kopieret som .env
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

### 3. Upload database

```bash
scp whoknows.db deploy@20.240.47.24:/opt/whoknows/data/whoknows.db
ssh deploy@20.240.47.24 "cd /opt/whoknows && docker compose restart whoknows"
```

### Resultat

- **https://huw.dk/** – virker med grøn lås.
- **http://huw.dk/** – omdirigerer til **https://huw.dk/**.
- **http://20.240.47.24** – omdirigerer til **https://huw.dk**.
- **https://20.240.47.24** – omdirigerer til **https://huw.dk**.

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

Database-filen (`whoknows.db`) overskrives **ikke** – den ligger i `/opt/whoknows/data/` og bevares mellem deploys.

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

## Database

### Upload jeres egen whoknows.db

Appen læser databasen fra `/opt/whoknows/data/whoknows.db`. I kan erstatte den:

```bash
# Fra din PC – kopier database til serveren
scp whoknows.db deploy@20.240.47.24:/opt/whoknows/data/whoknows.db

# Genstart containeren
ssh deploy@20.240.47.24 "cd /opt/whoknows && docker compose restart whoknows"
```

### Backup

```bash
scp deploy@20.240.47.24:/opt/whoknows/data/whoknows.db whoknows-backup.db
```

### Kør migration (hvis tabeller mangler)

```bash
ssh deploy@20.240.47.24
cd /opt/whoknows
docker compose exec whoknows sh -c "sqlite3 /app/data/whoknows.db < /app/migrations/001_init.sql"
```

### Tjek logs

```bash
ssh deploy@20.240.47.24 "cd /opt/whoknows && docker compose logs -f whoknows"
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
