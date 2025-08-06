#!/bin/bash
#
# Cloud Update Webhook Client
# Envoie des commandes sécurisées au service Cloud Update
#

set -e

# Configuration par défaut
DEFAULT_URL="http://localhost:9999/webhook"
DEFAULT_SECRET=""

# Couleurs pour l'output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Fonction d'aide
show_help() {
    cat << EOF
Usage: $0 [OPTIONS] ACTION

Actions disponibles:
  reinit          Réinitialiser cloud-init
  update          Mettre à jour le système
  upgrade         Mise à niveau complète du système
  reboot          Redémarrer le serveur
  shutdown        Éteindre le serveur
  restart         Redémarrer des services
  execute_script  Exécuter un script personnalisé

Options:
  -u, --url URL         URL du webhook (défaut: $DEFAULT_URL)
  -s, --secret SECRET   Secret HMAC (ou via CLOUD_UPDATE_SECRET)
  -m, --module MODULE   Module spécifique (optionnel)
  -c, --config KEY=VAL  Configuration additionnelle (peut être répété)
  -v, --verbose         Mode verbose
  -h, --help           Afficher cette aide

Exemples:
  # Mise à jour simple
  $0 --secret "mon-secret" update

  # Redémarrage avec configuration
  $0 --secret "mon-secret" --config "delay=30" reboot

  # Utiliser une variable d'environnement pour le secret
  export CLOUD_UPDATE_SECRET="mon-secret"
  $0 update

EOF
    exit 0
}

# Parser les arguments
URL="$DEFAULT_URL"
SECRET="$DEFAULT_SECRET"
ACTION=""
MODULE=""
CONFIG_ARGS=()
VERBOSE=0

while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--url)
            URL="$2"
            shift 2
            ;;
        -s|--secret)
            SECRET="$2"
            shift 2
            ;;
        -m|--module)
            MODULE="$2"
            shift 2
            ;;
        -c|--config)
            CONFIG_ARGS+=("$2")
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=1
            shift
            ;;
        -h|--help)
            show_help
            ;;
        *)
            ACTION="$1"
            shift
            ;;
    esac
done

# Vérifier l'action
if [ -z "$ACTION" ]; then
    echo -e "${RED}Erreur: Aucune action spécifiée${NC}"
    show_help
fi

# Récupérer le secret depuis l'environnement si pas fourni
if [ -z "$SECRET" ]; then
    SECRET="${CLOUD_UPDATE_SECRET:-}"
fi

if [ -z "$SECRET" ]; then
    echo -e "${RED}Erreur: Secret non fourni (utilisez --secret ou CLOUD_UPDATE_SECRET)${NC}"
    exit 1
fi

# Construire le payload JSON
TIMESTAMP=$(date +%s)
PAYLOAD='{"action":"'$ACTION'","timestamp":'$TIMESTAMP

# Ajouter le module si spécifié
if [ -n "$MODULE" ]; then
    PAYLOAD=$PAYLOAD',"module":"'$MODULE'"'
fi

# Ajouter la configuration si spécifiée
if [ ${#CONFIG_ARGS[@]} -gt 0 ]; then
    PAYLOAD=$PAYLOAD',"config":{'
    FIRST=1
    for config in "${CONFIG_ARGS[@]}"; do
        KEY="${config%%=*}"
        VALUE="${config#*=}"
        if [ $FIRST -eq 0 ]; then
            PAYLOAD=$PAYLOAD','
        fi
        PAYLOAD=$PAYLOAD'"'$KEY'":"'$VALUE'"'
        FIRST=0
    done
    PAYLOAD=$PAYLOAD'}'
fi

PAYLOAD=$PAYLOAD'}'

# Mode verbose
if [ $VERBOSE -eq 1 ]; then
    echo -e "${BLUE}URL: $URL${NC}"
    echo -e "${BLUE}Action: $ACTION${NC}"
    echo -e "${BLUE}Payload: $PAYLOAD${NC}"
fi

# Calculer la signature HMAC-SHA256
if command -v openssl >/dev/null 2>&1; then
    # Méthode OpenSSL (plus portable)
    SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" -binary | xxd -p -c 256)
elif command -v shasum >/dev/null 2>&1; then
    # Méthode macOS alternative
    SIGNATURE=$(echo -n "$PAYLOAD" | shasum -a 256 -h "$SECRET" | cut -d' ' -f1)
else
    echo -e "${RED}Erreur: Ni openssl ni shasum n'est installé${NC}"
    exit 1
fi

if [ $VERBOSE -eq 1 ]; then
    echo -e "${BLUE}Signature: sha256=$SIGNATURE${NC}"
fi

# Envoyer la requête
echo -e "${YELLOW}Envoi de l'action '$ACTION' au serveur...${NC}"

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$URL" \
    -H "Content-Type: application/json" \
    -H "X-Cloud-Update-Signature: sha256=$SIGNATURE" \
    -d "$PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

# Analyser la réponse
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ Succès!${NC}"
    if [ -n "$BODY" ]; then
        echo -e "${GREEN}Réponse: $BODY${NC}"
    fi
elif [ "$HTTP_CODE" = "401" ]; then
    echo -e "${RED}✗ Erreur d'authentification (401)${NC}"
    echo -e "${RED}Vérifiez votre secret${NC}"
    exit 1
elif [ "$HTTP_CODE" = "400" ]; then
    echo -e "${RED}✗ Requête invalide (400)${NC}"
    echo -e "${RED}Réponse: $BODY${NC}"
    exit 1
elif [ "$HTTP_CODE" = "405" ]; then
    echo -e "${RED}✗ Méthode non autorisée (405)${NC}"
    exit 1
else
    echo -e "${RED}✗ Erreur inattendue (HTTP $HTTP_CODE)${NC}"
    echo -e "${RED}Réponse: $BODY${NC}"
    exit 1
fi