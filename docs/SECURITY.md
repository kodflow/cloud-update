# 🔒 Sécurité du Cloud Update

## Vue d'ensemble

Cloud Update utilise **HMAC-SHA256** pour sécuriser tous les webhooks. Cela garantit que seules les requêtes autorisées
peuvent déclencher des actions sensibles comme :

- 🔄 **reinit** : Réinitialisation du cloud-init
- 🔧 **update** : Mise à jour du système
- 🔌 **reboot** : Redémarrage du serveur
- ⚡ **execute_script** : Exécution de scripts personnalisés

## 🛡️ Protection HMAC

### Comment ça fonctionne

1. **Secret partagé** : Un secret est configuré sur le serveur (`CLOUD_UPDATE_SECRET`)
2. **Signature** : Le client calcule une signature HMAC-SHA256 du body de la requête
3. **Vérification** : Le serveur vérifie que la signature correspond

### Format de la signature

```
X-Cloud-Update-Signature: sha256=<hex_signature>
```

## 📝 Exemples d'utilisation

### Bash/cURL

```bash
#!/bin/bash

# Configuration
SECRET="votre-secret-tres-securise"
URL="http://localhost:9999/webhook"

# Payload
PAYLOAD='{"action":"update","timestamp":'$(date +%s)'}'

# Calculer la signature HMAC-SHA256
SIGNATURE=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^SHA2-256= //')

# Envoyer la requête
curl -X POST "$URL" \
  -H "Content-Type: application/json" \
  -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
  -d "$PAYLOAD"
```

### Python

```python
import hmac
import hashlib
import json
import requests
import time

def send_secure_webhook(action, secret, url="http://localhost:9999/webhook"):
    # Créer le payload
    payload = {
        "action": action,
        "timestamp": int(time.time())
    }

    # Convertir en JSON
    payload_json = json.dumps(payload, separators=(',', ':'))

    # Calculer la signature HMAC
    signature = hmac.new(
        secret.encode('utf-8'),
        payload_json.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()

    # Envoyer la requête
    response = requests.post(
        url,
        data=payload_json,
        headers={
            'Content-Type': 'application/json',
            'X-Cloud-Update-Signature': f'sha256={signature}'
        }
    )

    return response

# Exemple d'utilisation
secret = "votre-secret-tres-securise"
response = send_secure_webhook("update", secret)
print(f"Status: {response.status_code}")
print(f"Response: {response.json()}")
```

### Go

```go
package main

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type WebhookRequest struct {
    Action    string `json:"action"`
    Timestamp int64  `json:"timestamp"`
}

func sendSecureWebhook(action, secret, url string) error {
    // Créer le payload
    payload := WebhookRequest{
        Action:    action,
        Timestamp: time.Now().Unix(),
    }

    // Convertir en JSON
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    // Calculer la signature HMAC
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(jsonData)
    signature := "sha256=" + hex.EncodeToString(h.Sum(nil))

    // Créer la requête
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Cloud-Update-Signature", signature)

    // Envoyer la requête
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode)
    return nil
}

func main() {
    secret := "votre-secret-tres-securise"
    sendSecureWebhook("update", secret, "http://localhost:9999/webhook")
}
```

### Node.js

```javascript
const crypto = require('crypto')
const http = require('http')

function sendSecureWebhook(action, secret, url = 'http://localhost:9999/webhook') {
  // Créer le payload
  const payload = {
    action: action,
    timestamp: Math.floor(Date.now() / 1000),
  }

  const payloadString = JSON.stringify(payload)

  // Calculer la signature HMAC
  const signature = crypto.createHmac('sha256', secret).update(payloadString).digest('hex')

  // Options de la requête
  const options = {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Cloud-Update-Signature': `sha256=${signature}`,
      'Content-Length': Buffer.byteLength(payloadString),
    },
  }

  // Envoyer la requête
  const req = http.request(url, options, (res) => {
    console.log(`Status: ${res.statusCode}`)

    res.on('data', (data) => {
      console.log(`Response: ${data}`)
    })
  })

  req.on('error', (error) => {
    console.error(`Error: ${error}`)
  })

  req.write(payloadString)
  req.end()
}

// Exemple d'utilisation
const secret = 'votre-secret-tres-securise'
sendSecureWebhook('update', secret)
```

## 🔑 Génération du secret

Pour générer un secret sécurisé :

```bash
# Méthode 1: OpenSSL
openssl rand -hex 32

# Méthode 2: /dev/urandom
cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1

# Méthode 3: Makefile
make generate-secret
```

## ⚠️ Bonnes pratiques

### ✅ À FAIRE

1. **Secret fort** : Utilisez un secret d'au moins 32 caractères
2. **HTTPS** : Utilisez HTTPS en production pour éviter l'interception
3. **Rotation** : Changez le secret régulièrement
4. **Stockage sécurisé** :
   - Permissions 600 sur `/etc/cloud-update/config.env`
   - Utilisez un gestionnaire de secrets (Vault, AWS Secrets Manager, etc.)
5. **Logs** : Ne loggez jamais le secret ou les signatures

### ❌ À NE PAS FAIRE

1. **Secret faible** : N'utilisez pas de mots de passe simples
2. **HTTP non sécurisé** : N'envoyez pas de webhooks en HTTP sur Internet
3. **Secret dans le code** : Ne hardcodez jamais le secret
4. **Partage du secret** : Ne partagez pas le secret via des canaux non sécurisés

## 🔍 Débogage

### Vérifier la signature manuellement

```bash
# Tester avec une signature valide
SECRET="test-secret"
PAYLOAD='{"action":"update","timestamp":1234567890}'
SIGNATURE=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^SHA2-256= //')

echo "Payload: $PAYLOAD"
echo "Signature: sha256=$SIGNATURE"
```

### Erreurs communes

| Erreur                 | Cause              | Solution                                     |
| ---------------------- | ------------------ | -------------------------------------------- |
| 401 Unauthorized       | Signature invalide | Vérifier le secret et le calcul de signature |
| 401 Unauthorized       | Header manquant    | Ajouter `X-Cloud-Update-Signature`           |
| 400 Bad Request        | JSON invalide      | Vérifier le format du payload                |
| 405 Method Not Allowed | Mauvaise méthode   | Utiliser POST uniquement                     |

## 🛠️ Configuration

### Variables d'environnement

```bash
# /etc/cloud-update/config.env
CLOUD_UPDATE_SECRET=votre-secret-tres-securise-ici
CLOUD_UPDATE_PORT=9999
CLOUD_UPDATE_LOG_LEVEL=info
```

### Permissions

```bash
# Sécuriser le fichier de configuration
sudo chmod 600 /etc/cloud-update/config.env
sudo chown root:root /etc/cloud-update/config.env
```

## 📊 Monitoring

### Logs d'authentification

Les tentatives d'authentification échouées sont loggées :

```bash
# Voir les logs
journalctl -u cloud-update -f | grep "Invalid signature"
```

### Métriques de sécurité

- Nombre de requêtes authentifiées avec succès
- Nombre de requêtes rejetées (401)
- Temps de réponse des validations HMAC

## 🚨 Réponse aux incidents

Si vous suspectez que votre secret a été compromis :

1. **Changez immédiatement le secret**

   ```bash
   # Générer un nouveau secret
   NEW_SECRET=$(openssl rand -hex 32)

   # Mettre à jour la configuration
   sudo sed -i "s/CLOUD_UPDATE_SECRET=.*/CLOUD_UPDATE_SECRET=$NEW_SECRET/" /etc/cloud-update/config.env

   # Redémarrer le service
   sudo systemctl restart cloud-update
   ```

2. **Vérifiez les logs** pour détecter des activités suspectes

3. **Mettez à jour tous les clients** avec le nouveau secret

4. **Considérez** une rotation régulière des secrets

## 📚 Références

- [RFC 2104 - HMAC](https://tools.ietf.org/html/rfc2104)
- [GitHub Webhooks Security](https://docs.github.com/en/developers/webhooks-and-events/securing-your-webhooks)
- [OWASP API Security](https://owasp.org/www-project-api-security/)
