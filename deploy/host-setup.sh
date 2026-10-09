#!/usr/bin/env bash
# One-time host bootstrap for the gateway. Idempotent; run as root:
#   DOMAIN=rdp.example.com EMAIL=you@example.com CF_API_TOKEN=... ./host-setup.sh
# Leaves existing UFW rules (SSH allowlist) untouched; only adds 443 from Cloudflare.
set -euo pipefail
: "${DOMAIN:?}" "${EMAIL:?}" "${CF_API_TOKEN:?}"
HERE=$(cd "$(dirname "$0")" && pwd)

apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
  nginx certbot python3-certbot-dns-cloudflare docker.io docker-compose-v2 ufw fail2ban curl >/dev/null
systemctl enable --now docker nginx fail2ban

# --- TLS via Cloudflare DNS-01 (no port 80 needed) ---
install -d -m 700 /etc/letsencrypt
umask 077
printf 'dns_cloudflare_api_token = %s\n' "$CF_API_TOKEN" > /etc/letsencrypt/cloudflare-webrdp.ini
umask 022
if [ ! -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
  certbot certonly --non-interactive --agree-tos -m "$EMAIL" \
    --dns-cloudflare --dns-cloudflare-credentials /etc/letsencrypt/cloudflare-webrdp.ini \
    --dns-cloudflare-propagation-seconds 30 -d "$DOMAIN"
fi
install -d /etc/letsencrypt/renewal-hooks/deploy
printf '#!/bin/sh\nsystemctl reload nginx\n' > /etc/letsencrypt/renewal-hooks/deploy/reload-nginx
chmod +x /etc/letsencrypt/renewal-hooks/deploy/reload-nginx

# --- Cloudflare ranges: real client IP + origin allowlist ---
CF4=$(curl -fsS https://www.cloudflare.com/ips-v4)
CF6=$(curl -fsS https://www.cloudflare.com/ips-v6)
{
  for r in $CF4 $CF6; do echo "set_real_ip_from $r;"; done
  echo "real_ip_header CF-Connecting-IP;"
} > /etc/nginx/conf.d/cloudflare-realip.conf
# realip rewrites $remote_addr, so test the TCP peer ($realip_remote_addr).
{
  echo 'geo $realip_remote_addr $webrdp_from_cf {'
  echo '    default 0;'
  for r in $CF4 $CF6; do echo "    $r 1;"; done
  echo '}'
} > /etc/nginx/conf.d/webrdp-cloudflare-geo.conf
install -d /etc/nginx/snippets
install -m 644 "$HERE/nginx/webrdp-proxy.conf" /etc/nginx/snippets/webrdp-proxy.conf
install -m 644 "$HERE/nginx/webrdp-limits.conf" /etc/nginx/conf.d/webrdp-limits.conf
sed "s/__DOMAIN__/$DOMAIN/g" "$HERE/nginx/site.conf.tmpl" > "/etc/nginx/sites-available/$DOMAIN"
ln -sf "/etc/nginx/sites-available/$DOMAIN" "/etc/nginx/sites-enabled/$DOMAIN"
rm -f /etc/nginx/sites-enabled/default
sed -i 's/# server_tokens off;/server_tokens off;/' /etc/nginx/nginx.conf
nginx -t
systemctl reload nginx

# --- Firewall: 443 from Cloudflare only; never touch existing SSH rules ---
ufw default deny incoming >/dev/null
ufw default allow outgoing >/dev/null
for r in $CF4 $CF6; do ufw allow proto tcp from "$r" to any port 443 comment cloudflare >/dev/null; done
ufw --force enable >/dev/null
ufw status numbered | head -40

# --- App directory ---
install -d -m 750 /opt/webrdp /opt/webrdp/db
install -m 640 "$HERE/docker-compose.yml" /opt/webrdp/docker-compose.yml
install -m 750 "$HERE/db/init-app-role.sh" /opt/webrdp/db/init-app-role.sh
[ -f /opt/webrdp/.env ] || { install -m 600 "$HERE/.env.example" /opt/webrdp/.env; echo "EDIT /opt/webrdp/.env"; }
echo "host setup done for $DOMAIN"
