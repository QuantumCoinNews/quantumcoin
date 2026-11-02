# Final onay kanıtları — TODO

Gerçek NFT'ler ve yetkili AI/admin onayları henüz oluşturulmadı. Bu klasör
QMA-0071'i kapatmaz. Son aşamada her katalog öğesi için `<catalogId>.json`
dosyası yetkili **kamusal imza kanıtını** içerecek; özel anahtar içermeyecek.

Şema ve tam bağ alanları:
`telegram_game/telegram_game/utils/nft_final_readiness_v1.py`.
İki ayrı Ed25519 rolü aynı statement'ı rol bazında alan ayrımlı imzalar.
Statement program/genesis, aday/alıcı/dönem, çekiliş receipt hash'i, katalog
byte hash'i, metadata ve görsel hash'lerini içerir. İçerik veya alıcı değişirse
eski onay kullanılamaz. Test fixture'ları gerçek onay olarak buraya kopyalanmaz.
