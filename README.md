# Cloud Update

[![Reference](https://pkg.go.dev/badge/github.com/kodflow/cloud-update.svg)](https://pkg.go.dev/github.com/kodflow/cloud-update)
[![Latest Stable Version](https://img.shields.io/github/v/tag/kodflow/cloud-update?label=version)](https://github.com/kodflow/cloud-update/releases/latest)
[![CI](https://img.shields.io/github/actions/workflow/status/kodflow/cloud-update/ci.yml?label=CI)](https://github.com/kodflow/cloud-update/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=bugs)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=code_smells)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=coverage)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=sqale_index)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=kodflow_cloud-update&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=kodflow_cloud-update)

**Cloud Update** est un agent léger de mise à jour système conçu pour les environnements cloud-init. Il permet
d'orchestrer des mises à jour système déclenchées par webhook avec support multi-distribution Linux.

## 🚀 Caractéristiques

- **Multi-distribution** : Support natif pour Alpine, Debian, Ubuntu, RHEL, CentOS, Fedora, Arch Linux
- **Webhook sécurisé** : Déclenchement via webhook avec signature HMAC-SHA256
- **Rate limiting** : Protection contre les abus avec limitation intelligente
- **Gestion des jobs** : File d'attente avec exécution séquentielle garantie
- **Observabilité** : Logs structurés JSON avec rotation automatique
- **Léger** : Binaire statique unique sans dépendances externes
- **Multi-architecture** : Support x86, ARM, PowerPC, S390x, MIPS, RISC-V

## 📦 Installation

### Via script d'installation

```bash
curl -sSL https://raw.githubusercontent.com/kodflow/cloud-update/main/install.sh | sudo sh
```

### Installation manuelle

```bash
# Télécharger le binaire pour votre architecture
wget https://github.com/kodflow/cloud-update/releases/latest/download/cloud-update-linux-amd64
sudo mv cloud-update-linux-amd64 /usr/local/bin/cloud-update
sudo chmod +x /usr/local/bin/cloud-update

# Installer le service systemd
sudo cloud-update install
sudo systemctl enable cloud-update
sudo systemctl start cloud-update
```

### Via cloud-init

```yaml
#cloud-config
packages:
  - curl

runcmd:
  - curl -sSL https://raw.githubusercontent.com/kodflow/cloud-update/main/install.sh | sh
  - systemctl enable cloud-update
  - systemctl start cloud-update
```

## 🔧 Configuration

### Variables d'environnement

```bash
# Secret pour la validation des webhooks (REQUIS)
CLOUD_UPDATE_SECRET="votre-secret-securise"

# Port d'écoute (défaut: 8080)
CLOUD_UPDATE_PORT="9999"

# Niveau de log (debug, info, warn, error)
CLOUD_UPDATE_LOG_LEVEL="info"

# Fichier de log (défaut: /var/log/cloud-update/cloud-update.log)
CLOUD_UPDATE_LOG_FILE="/var/log/cloud-update/cloud-update.log"

# Base de données (défaut: /var/lib/cloud-update/cloud-update.db)
CLOUD_UPDATE_DB_PATH="/var/lib/cloud-update/cloud-update.db"
```

### Fichier de configuration systemd

```ini
# /etc/systemd/system/cloud-update.service
[Unit]
Description=Cloud Update Agent
After=network.target

[Service]
Type=simple
User=root
Environment="CLOUD_UPDATE_SECRET=votre-secret"
Environment="CLOUD_UPDATE_PORT=9999"
Environment="CLOUD_UPDATE_LOG_LEVEL=info"
ExecStart=/usr/local/bin/cloud-update serve
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## 📡 Utilisation

### Démarrer le serveur

```bash
# Avec systemd
sudo systemctl start cloud-update

# En direct
CLOUD_UPDATE_SECRET="secret" cloud-update serve

# Avec Docker
docker run -d \
  -p 8080:9999 \
  -e CLOUD_UPDATE_SECRET="secret" \
  ghcr.io/kodflow/cloud-update:latest
```

### Déclencher une mise à jour via webhook

```bash
# Générer la signature HMAC
PAYLOAD='{"action":"update","timestamp":'$(date +%s)'}'
SECRET="votre-secret"
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)

# Envoyer le webhook
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
  -d "$PAYLOAD" \
  http://localhost:9999/webhook
```

### Exemple avec GitHub Actions

```yaml
name: Trigger Cloud Update

on:
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger update webhook
        env:
          SECRET: ${{ secrets.CLOUD_UPDATE_SECRET }}
          TARGET: ${{ secrets.TARGET_SERVER }}
        run: |
          PAYLOAD='{"action":"update","timestamp":'$(date +%s)'}'
          SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)

          curl -X POST \
            -H "Content-Type: application/json" \
            -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
            -d "$PAYLOAD" \
            "$TARGET/webhook"
```

### Exemple Python

```python
import hmac
import hashlib
import json
import time
import requests

def trigger_update(url, secret):
    payload = {
        "action": "update",
        "timestamp": int(time.time())
    }

    # Générer la signature
    payload_str = json.dumps(payload, separators=(',', ':'))
    signature = hmac.new(
        secret.encode(),
        payload_str.encode(),
        hashlib.sha256
    ).hexdigest()

    # Envoyer la requête
    response = requests.post(
        f"{url}/webhook",
        json=payload,
        headers={
            "X-Cloud-Update-Signature": f"sha256={signature}"
        }
    )

    return response.json()

# Utilisation
result = trigger_update("http://server.example.com:9999", "secret")
print(f"Job ID: {result['job_id']}")
```

## 🔍 Endpoints API

### `GET /health`

Vérification de l'état du service.

```bash
curl http://localhost:9999/health
# {"status":"healthy","timestamp":1234567890}
```

### `POST /webhook`

Déclenche une action (mise à jour, reboot, etc.).

**Headers requis:**

- `Content-Type: application/json`
- `X-Cloud-Update-Signature: sha256=<signature>`

**Body:**

```json
{
  "action": "update|reboot|cloud-init",
  "timestamp": 1234567890
}
```

**Réponse:**

```json
{
  "status": "accepted",
  "job_id": "job_123456",
  "message": "Update job queued"
}
```

### `GET /metrics`

Métriques Prometheus (si activé).

## 🏗️ Architecture

```text
cloud-update/
├── src/
│   ├── cmd/cloud-update/       # Point d'entrée
│   ├── internal/
│   │   ├── application/         # Handlers HTTP
│   │   ├── domain/             # Logique métier
│   │   ├── infrastructure/     # Services système
│   │   └── version/            # Gestion des versions
│   └── test/
│       └── e2e/                # Tests end-to-end
├── scripts/                    # Scripts de build et CI
├── BUILD.bazel                # Configuration Bazel
└── Makefile                   # Commandes de développement
```

## 🛠️ Développement

### Prérequis

- Go 1.24+
- Bazel 7.0+
- Docker (pour les tests E2E)

### Build

```bash
# Build pour la plateforme courante
make build

# Build pour toutes les architectures Linux
make build/all

# Build avec Bazel directement
bazel build //src/cmd/cloud-update:cloud-update
```

### Tests

```bash
# Tous les tests avec vérifications qualité
make test

# Tests unitaires uniquement
make test/unit

# Tests E2E (nécessite Docker)
make test/e2e

# Tests rapides sans lint
make test/quick
```

### Qualité du code

```bash
# Formatage
make format

# Linting
make lint

# Scan de sécurité
make security

# Toutes les vérifications
make quality
```

## 📊 Performances

- **Démarrage** : < 100ms
- **Empreinte mémoire** : ~15MB au repos
- **Latence webhook** : < 10ms (p99)
- **Concurrence** : 10,000 req/s avec rate limiting

## 🔒 Sécurité

- Validation HMAC-SHA256 sur tous les webhooks
- Rate limiting par IP avec LRU intelligent
- Exécution séquentielle des jobs (protection cloud-init)
- Pas de shell injection (commandes prédéfinies)
- Logs sans données sensibles
- Support privilege escalation (sudo/doas/su)

## 🤝 Contribution

Les contributions sont les bienvenues ! Consultez [CONTRIBUTING.md](CONTRIBUTING.md) pour les directives.

```bash
# Fork le projet
git clone https://github.com/your-username/cloud-update.git
cd cloud-update

# Créer une branche
git checkout -b feat/ma-fonctionnalite

# Faire vos changements et tester
make test

# Commit avec message conventionnel
git commit -m "feat: ajouter support pour ..."

# Pousser et créer une PR
git push origin feat/ma-fonctionnalite
```

## 📝 License

MIT - Voir [LICENSE](LICENSE) pour plus de détails.

## 🙏 Crédits

Développé avec ❤️ par [Kodflow](https://github.com/kodflow)
