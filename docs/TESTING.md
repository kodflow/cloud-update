# Testing Guide

## 🧪 Test Suite Overview

Le projet cloud-update dispose d'une suite de tests complète qui peut être exécutée localement ou via GitHub Actions.

## 📋 Tests Locaux

### Tests Rapides
```bash
# Suite complète (lint, format, security, unit tests)
make tests

# Tests unitaires uniquement
go test -v ./src/...

# Tests avec coverage
make coverage
```

### Tests E2E
```bash
# Tous les tests E2E (nécessite Docker)
make tests-e2e

# Test E2E pour une distribution spécifique
make e2e-distro DISTRO=alpine
make e2e-distro DISTRO=ubuntu
make e2e-distro DISTRO=rockylinux
```

## 🎬 Tests GitHub Actions en Local avec Act

[Act](https://github.com/nektos/act) permet d'exécuter les workflows GitHub Actions localement dans Docker.

### Installation d'Act

```bash
# macOS avec Homebrew
brew install act

# Linux/macOS avec curl
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Windows avec Chocolatey
choco install act-cli
```

### Utilisation

#### Interface Interactive
```bash
# Lance le menu interactif
make test-github-local

# Ou directement
./scripts/test-github-locally.sh
```

Options disponibles :
1. **Full CI Pipeline** : Exécute tous les jobs
2. **Quality checks** : Lint, format, security uniquement
3. **Unit tests** : Tests unitaires uniquement
4. **E2E tests** : Tests end-to-end uniquement
5. **Build validation** : Validation du build uniquement
6. **List workflows** : Liste tous les workflows disponibles
7. **Dry run** : Montre ce qui serait exécuté sans le faire
8. **Custom job** : Sélection manuelle d'un job
9. **PR simulation** : Simule une pull request

#### Commandes Directes
```bash
# Lister les workflows et jobs disponibles
make test-github-list
act -l

# Exécuter un job spécifique
make test-github-job JOB=quality
make test-github-job JOB=test
make test-github-job JOB=e2e

# Exécuter tout le workflow CI
act push --workflows .github/workflows/ci.yml

# Simuler une pull request
act pull_request

# Dry run (voir ce qui serait exécuté)
act push --dryrun

# Utiliser une image Docker spécifique
act -P ubuntu-latest=catthehacker/ubuntu:act-latest
```

### Configuration Act

Le fichier `.actrc` contient la configuration par défaut :
- Images Docker optimisées pour act
- Workflow par défaut (ci.yml)
- Réutilisation des conteneurs pour la performance
- Support de Docker BuildKit

### Secrets et Variables

Pour tester avec des secrets :
```bash
# Créer un fichier .secrets
echo "GITHUB_TOKEN=your-token" > .secrets
echo "CLOUD_UPDATE_SECRET=test-secret" >> .secrets

# Utiliser les secrets
act push --secret-file .secrets
```

Pour les variables d'environnement :
```bash
# Créer un fichier .env
echo "E2E_BASE_URL=http://localhost:9999" > .env
echo "E2E_SECRET=test-secret" >> .env

# Utiliser les variables
act push --env-file .env
```

## 🔍 Débuggage

### Mode Verbose
```bash
# Act en mode verbose
act push -v

# Tests Go en mode verbose
go test -v ./src/...
```

### Conteneur Interactif
```bash
# Garder le conteneur après l'exécution pour debug
act push --rm=false

# Shell interactif dans le conteneur
act push --container-architecture linux/amd64 -s GITHUB_TOKEN=fake
```

## 📊 Métriques de Test

### Coverage
```bash
# Générer le rapport de coverage
make coverage

# Voir le rapport HTML
open coverage.html
```

### Benchmarks
```bash
# Lancer les benchmarks
make bench

# Benchmark spécifique
go test -bench=BenchmarkWebhookHandler ./src/...
```

## 🚀 CI/CD Pipeline

Le pipeline GitHub Actions comprend 6 phases :

1. **Quality** : Lint, security scan, format check
2. **Test** : Tests unitaires sur Ubuntu et macOS
3. **E2E** : Tests sur Alpine, Ubuntu, Rocky Linux
4. **Validate** : Build multi-plateforme (PR uniquement)
5. **Release** : Release automatique (main branch)
6. **Status** : Rapport final

### Déclencher le Pipeline

- **Push sur main** : Pipeline complet avec release
- **Pull Request** : Pipeline sans release
- **Tag** : Release manuelle
- **`[skip ci]`** : Skip le pipeline
- **`[skip release]`** : Skip uniquement la release

## 🐛 Troubleshooting

### Act ne fonctionne pas
- Vérifier que Docker est lancé
- Vérifier l'espace disque disponible
- Nettoyer les images Docker : `docker system prune -a`

### Tests E2E échouent
- Vérifier que les ports 9991-9997 sont libres
- Vérifier les logs Docker : `docker logs cloud-update-test`
- Reconstruire les images : `docker compose -f src/test/e2e/docker-compose.yml build --no-cache`

### GitHub Actions échoue mais pas en local
- Différences d'environnement (OS, versions Go)
- Secrets/variables manquants
- Permissions de fichiers différentes
- Utiliser act pour reproduire l'environnement exact

## 📚 Ressources

- [Act Documentation](https://github.com/nektos/act)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Docker Documentation](https://docs.docker.com/)