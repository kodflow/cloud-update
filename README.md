# Cloud Update

Un service webhook sécurisé en Go pour redéclencher des actions cloud-init et des opérations système.

## Fonctionnalités

- **Webhook HTTP sécurisé** : Endpoint protégé par signature HMAC SHA256
- **Multi-distribution** : Support pour Alpine, Debian, Ubuntu, RHEL, CentOS, Fedora, SUSE, Arch
- **Actions supportées** :
  - `reinit` : Redéclenche cloud-init
  - `reboot` : Redémarre le système (avec délai de sécurité)
  - `update` : Met à jour le système avec le gestionnaire de paquets approprié
- **Multi-init** : Compatible avec systemd, OpenRC (Alpine), SysVinit
- **Configuration via variables d'environnement**
- **Logging détaillé**
- **Health check endpoint**

## Installation

### Installation automatique

Un script d'installation automatique est fourni pour simplifier le déploiement :

```bash
# Rendre le script exécutable
chmod +x install.sh

# Lancer l'installation
sudo ./install.sh
```

Le script détecte automatiquement :
- La distribution Linux (Alpine, Debian, Ubuntu, etc.)
- Le système d'init (systemd, OpenRC, SysVinit)

### Installation manuelle

#### Compilation

```bash
go build -o cloud-update cmd/cloud-update/main.go
```

#### Configuration

1. Copiez le fichier de configuration :

```bash
sudo mkdir -p /etc/cloud-update
sudo cp config.example.env /etc/cloud-update/config.env
```

2. Éditez la configuration :

```bash
sudo nano /etc/cloud-update/config.env
```

3. Générez une clé secrète sécurisée :

```bash
openssl rand -hex 32
```

#### Installation selon le système d'init

##### Systemd (Debian, Ubuntu, RHEL, etc.)

```bash
sudo mkdir -p /opt/cloud-update
sudo cp cloud-update /opt/cloud-update/
sudo cp init/systemd/cloud-update.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable cloud-update
sudo systemctl start cloud-update
```

##### OpenRC (Alpine Linux)

```bash
sudo mkdir -p /opt/cloud-update
sudo cp cloud-update /opt/cloud-update/
sudo cp init/openrc/cloud-update /etc/init.d/
sudo chmod 755 /etc/init.d/cloud-update
sudo rc-update add cloud-update default
sudo rc-service cloud-update start
```

##### SysVinit

```bash
sudo mkdir -p /opt/cloud-update
sudo cp cloud-update /opt/cloud-update/
sudo cp init/sysvinit/cloud-update /etc/init.d/
sudo chmod 755 /etc/init.d/cloud-update
sudo update-rc.d cloud-update defaults
sudo service cloud-update start
```

## Utilisation

### Health Check

```bash
curl http://localhost:9999/health
```

### Déclencher une action

```bash
# Générer la signature HMAC
PAYLOAD='{"action":"reinit","timestamp":1691234567}'
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "your-secret-key" | sed 's/.* //')

# Envoyer la requête
curl -X POST http://localhost:9999/webhook \
  -H "Content-Type: application/json" \
  -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

### Actions disponibles

#### Reinit (cloud-init)
```json
{
  "action": "reinit",
  "timestamp": 1691234567
}
```

#### Reboot
```json
{
  "action": "reboot",
  "timestamp": 1691234567
}
```

#### Update
```json
{
  "action": "update",
  "timestamp": 1691234567
}
```

## Sécurité

- **Authentification HMAC** : Toutes les requêtes webhook doivent être signées
- **Variables d'environnement** : Configuration sensible via variables d'env
- **Validation** : Vérification des payloads JSON et signatures
- **Logging** : Traçabilité de toutes les actions

## Configuration

### Variables d'environnement

| Variable | Description | Défaut |
|----------|-------------|---------|
| `CLOUD_UPDATE_PORT` | Port d'écoute HTTP | `9999` |
| `CLOUD_UPDATE_SECRET` | Clé secrète HMAC (obligatoire) | - |
| `CLOUD_UPDATE_LOG_LEVEL` | Niveau de log | `info` |

## Développement

### Tests

```bash
go test ./...
```

### Build

```bash
go build -o cloud-update cmd/cloud-update/main.go
```

## Distributions supportées

Le service a été testé sur les distributions suivantes :

- **Alpine Linux** (OpenRC)
- **Debian / Ubuntu** (systemd)
- **RHEL / CentOS / Fedora** (systemd)
- **SUSE / openSUSE** (systemd)
- **Arch Linux** (systemd)

Les gestionnaires de paquets suivants sont supportés pour l'action `update` :
- `apk` (Alpine)
- `apt` / `apt-get` (Debian/Ubuntu)
- `yum` / `dnf` (RHEL/CentOS/Fedora)
- `zypper` (SUSE/openSUSE)
- `pacman` (Arch)

## Désinstallation

Un script de désinstallation est fourni :

```bash
chmod +x uninstall.sh
sudo ./uninstall.sh
```

## Logs

Les logs sont accessibles selon le système d'init :

### Systemd

```bash
sudo journalctl -u cloud-update -f
```

### OpenRC (Alpine)

```bash
tail -f /var/log/cloud-update.log
```

### SysVinit

```bash
tail -f /var/log/syslog | grep cloud-update
```
