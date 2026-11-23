# QMA-0066–0073 üretim öncesi hazırlık

Bu klasör **taslak hazırlık alanıdır**. Gerçek NFT görselleri ve gerçek AI/admin
onayları henüz oluşturulmadı. Kullanıcı kararıyla QMA-0071 son aşamaya kadar
**AÇIK** kalır. Buradaki şablonlar aktivasyon veya mint yetkisi vermez.

Mevcut 12 taslak metadata `nft/catalog/metadata/` altında korunur. 2026 dosya
adları gerçek program açılış tarihi değildir. Gerçek dönemler node'un doğrulanmış
programındaki `gameActivatedAt` değerinden türeyen `awardPeriods` listesidir.

## İş sırası

1. Doğrulanmış Telegram kimliği ve sunucu oyun faaliyetiyle aday oluşturulur;
   yerel danışma incelemesi ve admin aday onayı tamamlanır.
2. Kullanıcı `/api/nft/wallet/challenge` çağrısından gelen `signingBytesHex`
   baytlarını secp256k1/SHA256 ile imzalar ve `/api/nft/wallet/prove` yoluna verir.
   İmza ASN.1 DER, açık anahtar 65 bayt `04||X||Y`, taşıma küçük harf hex'tir.
   Cüzdan adresi mevcut QuantumCoin sözleşmesindeki HASH160(X||Y) ile eşleşir.
   İmzalama desteği cüzdan arayüzüne henüz eklenmedi; özel anahtar sunucuya verilmez.
3. Dönem kapandıktan sonra `/api/admin/nft/monthly/freeze-pool` çağrılır.
   Havuzun her üyesinin cüzdan kanıtı freeze anından önce tüketilmiş olmalıdır.
   Eksik kanıtı olan havuz için taahhüt/çekiliş ve mint yetkisi üretilmez.
4. Yerel olarak pinlenmiş güven yapılandırmasıyla `draw-commitment` okunur.
   `commitmentHash = SHA256("QC_NFT_DRAW_COMMITMENT_V1\0" || canonicalJSON)`.
   Kanonik JSON: sıralı anahtarlar, boşluksuz ayraçlar, ASCII escape, NaN yok.
5. `publish-draw-commitment`, kalıcı outbox üzerinden bağımsız dış yayın tanığına
   gönderir ve aynı kamusal kaydı tekrar okuyarak doğrular. Bağımsız tanık bu özeti
   herkesin doğrulayabildiği HTTPS kaydına
   bağlar. İmzalı `QC_NFT_PUBLICATION_WITNESS_V1` statement'ı dış servisin yayın
   zamanını ve URL'sini içerir; yerel saat veya admin beyanı bunun yerine geçmez.
   `register-draw-commitment` hem yayın hem kayıt zamanını sabit beacon turundan
   önce doğrular. Tanık servisi kuruluma hazırdır; bağımsız sunucu/işletmeci ve
   gerçek anahtarlar henüz seçilmedi. Kurulum için `INDEPENDENT-WITNESS.md` okunur.
6. `draw-from-beacon`, pinlenmiş ağ bilgisini ve yalnız taahhütteki sabit turu
   HTTPS üzerinden alır. `pedersen-bls-unchained` ve Quicknet
   `bls-unchained-g1-rfc9380` şemalarının gerçek BLS imzası ve SHA256(signature)
   doğrulanır. `latest`, farklı tur, manuel seçim veya imzasız fallback yoktur.
   Ağ kimliği, anahtar, genesis ve tur süresi yerel policy'de pinlidir.
7. 12 gerçek görsel/metadata hazır olduğunda dosya byte hashleri ve katalog
   tekilliği doğrulanır. Her ödül için **ayrı** yetkili AI ve admin Ed25519
   imzaları program/dönem/alıcı/çekiliş/katalog/görsel/metadata bağını onaylar.
   Yerel danışma puanı, status string'i veya kamuya açık approval tokenı imzalı
   final onay yerine geçmez.
8. Mevcut node yetkililerinin imzaladığı `signedAward`, `controlled-mint`
   köprüsüne verilir. Oyun, `PENDING` intent'i ağ çağrısından önce kalıcılaştırır.
   Çekiliş ve final onay hashleri intent bağının parçasıdır; ağ çağrısından
   sonra içerik/kanıt yeniden doğrulanır. Oyun ancak node'un kalıcı kanonik
   receipt'ini doğruladıktan sonra `MINTED_CONFIRMED` yazar.
9. Teyitli mint daha sonra node'da 404 olursa `MINT_RECONCILIATION_REQUIRED`
   kaydedilir. Alıcı değiştirilmez, otomatik yeniden mint gönderilmez. Aynı
   kayıt kanonik zincire geri dönerse salt receipt sorgusuyla yeniden teyit edilir.

## Gerçek girdiler — tamamlanmadı

QMA-0070, kullanıcının bağımsız sunucu henüz olmadığı için istediği **kuruluma
hazır teknik teslim** kapsamındadır. Canlı bağımsız yayın henüz yapılmadı.
QMA-0071'in teknik altyapısı tamamlandı; gerçek varlık/onay koşulları açık kalır.

`GET /api/admin/nft/monthly/catalog-draft`, doğrulanmış aktivasyondan türeyen
12 gerçek dönem için taslak döndürür; dosya yazmaz. `POST final-readiness`,
aynı admin route kökü altında eksik varlık/onay aşamasını bildirir.
`POST final-approval-statement`, bütün final dosyalar ve çekiliş doğrulandıktan
sonra AI/admin için imzalanacak baytları hazırlar; imza veya mint yetkisi üretmez.
Final anahtarlar ve imzalar yerel güven yapılandırmasıyla ayrıca doğrulanır.

- TODO: 12 özgün final görsel ve metadata byte'ları; AI/admin içerik değerlendirmesi.
- TODO: Gerçek program aktivasyonu ve node'un yetkili imzacı/quorum seçimi.
- TODO: Bağımsız yayın tanığı, erişilebilir kamusal kayıt ve pinlenmiş tanık anahtarı.
- TODO: Yetkilendirilmiş drand ağ kimliği/anahtarı/zaman parametreleri.
- TODO: Ayrı AI/admin final onay anahtarları ve gerçek onay dosyaları.
- TODO: Kullanıcı cüzdan arayüzünde challenge byte'larının onaylanıp imzalanması.

`evidence-policy.example.jsonc` yalnız açıklamalı taslaktır; çalışan yapılandırma
değildir. `QC_NFT_EVIDENCE_POLICY_PATH` yerel **mutlak** JSON yolunu göstermelidir.
Eksik dosya, yanlış anahtar veya eksik bağımlılık işlemi durdurur. HTTP isteği
policy, anahtar veya node güven kökü belirleyemez.

## Tekrar çalıştırma

Proje Python venv'ine `telegram_game/requirements-nft-evidence.txt` kurulur;
sistem Python'u değiştirilmez. Go sözleşmesini ürettikten sonra aynı mutlak
JSON yolu Python testine verilir:

```powershell
$env:QMA_NFT_LIFECYCLE_CONTRACT_PATH = Join-Path $env:TEMP 'qma-nft-test-contract.json'
go test ./node -run '^TestNFTVerifiedLifecycleContractV1$' -count=1
# Proje venv'indeki python executable'ı ile:
python telegram_game/tests/test_nft_verified_lifecycle_v1.py
```

Test yalnız geçici SQLite/zincir, test anahtarları ve test görsel byte'ları
kullanır. Çalışan Go node'u ile mint/receipt testi dahildir. Başarılı test gerçek
ürün varlığı/onayı oluşturmaz ve 0071'i kapatmaz. İnsan tekilliği/KYC garantisi
yoktur; bir kişinin çoklu Telegram hesabı/cüzdan kullanma riski devam eder.

Eski miner NFT/yıllık 100 QC ödülü açılmaz. Ayrı QC arzı veya sahiplik DB'si
oluşturulmaz. Kanonik NFT sahipliği kabul edilmiş zincir kayıtlarından okunur.
