#!/usr/bin/env bash
# Bir marta ishlatiladi — serverga birinchi marta qo'yilganda
set -euo pipefail

# .env dan o'zgaruvchilarni yukla
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

DOMAIN="${DOMAIN:?Xato: .env da DOMAIN o'rnatilmagan}"
EMAIL="${EMAIL:?Xato: .env da EMAIL o'rnatilmagan}"

CERT_DIR="./certbot/conf/live/$DOMAIN"
WWW_DIR="./certbot/www"

echo "🔐 SSL sozlash: $DOMAIN"
echo ""

mkdir -p "$CERT_DIR" "$WWW_DIR"

# 1-qadam: vaqtincha sertifikat — nginx ishga tushishi uchun
if [ ! -f "$CERT_DIR/fullchain.pem" ]; then
  echo "→ Vaqtincha self-signed sertifikat yaratilmoqda..."
  openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
    -keyout "$CERT_DIR/privkey.pem" \
    -out    "$CERT_DIR/fullchain.pem" \
    -subj   "/CN=$DOMAIN" 2>/dev/null
  echo "   ✓ Tayyor"
fi

# 2-qadam: nginx va scorepoint ishga tushirish
echo "→ Servislar ishga tushirilmoqda..."
docker compose up -d nginx scorepoint
sleep 3

# 3-qadam: Let's Encrypt dan haqiqiy sertifikat olish
echo "→ Let's Encrypt sertifikati olinmoqda ($DOMAIN)..."
docker compose run --rm certbot certonly \
  --webroot --webroot-path /var/www/certbot \
  --email "$EMAIL" \
  --agree-tos --no-eff-email \
  --force-renewal \
  -d "$DOMAIN"

# 4-qadam: nginx ni yangi sertifikat bilan qayta yuklash
echo "→ Nginx qayta yuklanmoqda..."
docker compose exec nginx nginx -s reload

# 5-qadam: barcha servislarni ishga tushirish
docker compose up -d

echo ""
echo "✅ Muvaffaqiyatli!"
echo "   Scoreboard:  https://$DOMAIN"
echo "   OBS URL:     https://$DOMAIN"
echo ""
echo "   Keyingi marta ishga tushirish: docker compose up -d"
