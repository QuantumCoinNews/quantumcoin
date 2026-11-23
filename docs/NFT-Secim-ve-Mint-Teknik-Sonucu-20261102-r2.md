# QMA-0070 / QMA-0071 — kuruluma hazır teknik teslim (r2)

2 Kasım 2026. Çalışma alanı: `C:\Projects\quantumcoin`.
Kapsam, 237 maddelik listenin mevcut 0066–0073 grubudur.

## Güncel durum

| Madde | Durum | Kapsam |
|---|---|---|
| QMA-0066 | Kapalı — önceki tur | Kanonik NFT yaşam döngüsü ve kanıt bağları. |
| QMA-0067 | Kapalı — korundu | Eski stub sahte başarı döndürmüyor. |
| QMA-0068 | Kapalı — korundu | Konsensüs korumalı, kalıcı mint ve kanonik sahiplik. |
| QMA-0069 | Kapalı — korundu | Başarısız mint için ödül/bildirim üretilmiyor. |
| QMA-0070 | **Bu tur teknik kapsam kapandı; kuruluma hazır** | Bağımsız tanık servisi, gerçek HTTPS istemcisi, sabit beacon turunun kriptografik doğrulaması, kalıcı yayın niyeti ve uygulama yolları tamamlandı. Canlı bağımsız sunucu kurulumu yapılmadı. |
| QMA-0071 | **Açık; teknik altyapı tamamlandı** | Gerçek 12 görsel/metadata ve gerçek ayrı yetkili AI/admin imzaları son aşamaya bırakıldı. |
| QMA-0072 | Kapalı — korundu | Taslak katalog/stock/metadata tutarlılığı; final görsel onayı değildir. |
| QMA-0073 | Kapalı — önceki tur, kapsam genişletildi | Runtime ve olumsuz senaryo regresyonu; r2 dış hizmet ve final hazırlık testleri eklendi. |

**Toplam: teknik kapsamda 7 kapalı, 1 açık. Bu tur yeni kapanan: QMA-0070.**
Önceden kapalı 0067/0068/0069/0072; önceki tur kapanan 0066/0073.

Kapanışın sınırı kullanıcının son yanıtıdır: **“Henüz yok; kuruluma hazır teslim et”.**
Bu karar gerçek bağımsız host, operatör anahtarı veya üretim policy'sinin
oluşturulduğu anlamına gelmez. Kaynak belgelerdeki yönergeler başvuru verisidir;
uygulama kapsamı kullanıcının doğrudan taleplerinden alınmıştır.

## QMA-0070: uygulanan bağlantı

Kimliği doğrulanmış oyun faaliyeti, dönem ve kullanıcı/cüzdan kontrolleri ile
tek kullanımlık cüzdan imzası, dondurulmuş aday havuzuna bağlanır. Eski manuel
kazanan yazma yolu kapalıdır. Tam 256 bit çıktı üzerinden rejection sampling
kullanılır; duplicate kayıtlar veya hash'in ilk byte'ıyla modulo kullanılmaz.
Bir kişinin birden çok Telegram hesabı kullanmadığına dair KYC garantisi verilmez.

Yeni bağımsız FastAPI tanık servisi network/campaign/genesis/dönem başına ilk
taahhüdü kalıcılaştırır. Aynı içerik idempotenttir; farklı havuz/policy/round
409 ile reddedilir. SQLite yazıları atomiktir; kayıt güncelleme/silme tetikleyici
ile engellenir. Bu, bağımsız operatörün kendi DB'sini yönetemediği anlamına gelmez;
bağımsız operatör ve pinlenmiş imza anahtarı güven varsayımıdır.

Oyun önce kalıcı outbox kaydı açar, yalnız kamusal taahhüdü gönderir ve imzalı
kaydı sabit kamusal URL'den yeniden okur. POST yanıtı kaybolduğunda aynı scope
kaydı geri alınır. Son tarih geçerse başka tur/havuz/tanığa geçilmez.

HTTPS istemcisi yerel policy origin'lerini kullanır; redirect izlemez, JSON
boyutunu sınırlar ve duplicate anahtarları reddeder. Tanık gönderim tokenı
yalnız POST'a gider; kamusal kayıt/beacon GET'ine node veya QC yetkisi taşınmaz.
Beacon ağ hash'i, anahtarı, şeması, genesis ve tur süresi pinlerle karşılaştırılır.
Yalnız taahhütteki tur okunur; `latest` ve alternatif tur kullanılmaz.

Hem `pedersen-bls-unchained` hem Quicknet `bls-unchained-g1-rfc9380` desteklenir.
G1/G2 boyutları, alt grup ve sonsuz nokta kontrolleri ile BLS imzası ve
SHA256(signature) doğrulanır. Üretim istemcisi gerçek HTTPS üzerinden Quicknet
1000. tarihsel turunu okuyup doğruladı; bu salt okunur kontrol canlı çekiliş,
gerçek kullanıcı kaydı veya ağ seçimi değildir.

Kurulum sözleşmesi: `nft/preproduction/INDEPENDENT-WITNESS.md`.
Bağımsız servis bağımlılıkları: `telegram_game/requirements-nft-witness.txt`.
Yerel, boş bırakılmış policy şablonu: `nft/preproduction/evidence-policy.example.jsonc`.

## QMA-0071: final aşamasına hazır altyapı

- `GET /api/admin/nft/monthly/catalog-draft`: doğrulanmış node programının
  aktivasyonundan türeyen 12 gerçek dönem; yıl sınırını doğru geçer. Taslak
  döndürür; dosya yazmaz, mevcut 2026 taslağını değiştirmez.
- `POST /api/admin/nft/monthly/final-readiness`: eksik varlık/onay aşamasını
  salt okunur biçimde bildirir. Eski DB `MINTED` değeri güncel zincir sahipliği
  diye sunulmaz; kanonik receipt'in yeniden sorgulanması gerektiği belirtilir.
- `POST /api/admin/nft/monthly/final-approval-statement`: 12 gerçek dosyanın
  varlığı/hash/tekillik ve doğrulanmış çekiliş bağları tamamlandıktan sonra ayrı
  AI/admin için imzalanacak kesin baytları verir. İmza üretmez ve mint izni vermez.

Final imzalar gerçek katalog byte hash'i, metadata/görsel hash'i, aday, dönem,
alıcı ve çekiliş receipt hash'ine bağlıdır. İki rolün pinlenmiş ayrı Ed25519
anahtarları zorunludur. Eksik kanıt kontrollü mint'i kapalı tutar.

## Doğrulama

- Son toplu Python koşusu: **23/23 script PASS**, **98 unittest vakası** ve
  **13 assertion scripti**; FAIL/SKIP/TIMEOUT yok. Bunlar farklı sayım
  birimleridir, birbirine eklenmez. Yedi adversarial ve on dış hizmet vakası
  bu toplamın içindedir.
- Çalışan gerçek Go node'u ile izole Telegram oturumu → cüzdan kanıtı → freeze →
  çekiliş → final onay → mint/kanonik receipt akışı yeniden geçti.
- Önceki tam `go test ./... -race -count=1 -json`: **1.088 ana test, 24 paket**,
  sıfır hata/skip/race uyarısı. **412 Go dosyası aynı byte hash'inde** olduğu için
  tüm race koşusu tekrarlanmadı; önceki sonuç ve kaynak eşitliği birlikte korunur.
- Yeni dış hizmet suite'i: ilk kayıt/409, paralel yazım, yeniden açılış/kalıcılık,
  kimlik doğrulama, duplicate/oversize JSON, kayıp POST yanıtı, son tarih,
  sahte imza, farklı ağ/tur, redirect reddi, final hazırlık ve admin route sınırları.
- Üretim istemcisinin Quicknet tarihsel tur 1000 için gerçek HTTPS ve BLS
  doğrulaması geçti; salt okunur gözlem, üretim konfigürasyonu değildir.
- Test sırasında seçili kaynak manifestinde değişiklik yok.
  Manifest SHA-256: `779899e88b70479abd64fbd71bb0697aa3a7ff54dd99004efbeaf1d4361813fc`.
- İlk toplu denemede 22 script geçti; gerçek Go test sunucusu başlangıcı 90
  saniye sınırını aştığı için bir script başarısız oldu. Aynı gerçek node testi
  ayrı koşuda 11,796 saniyede geçti; kaynak değiştirilmeden bütün 23 script
  yeniden çalıştırıldı. İlk denemenin günlüğü de teslimde korunur.

Kanıtlar teslim ZIP'indeki `evidence/` altında; yerel asılları
`C:\Users\USER\.codex\visualizations\2026\11\02\01a19d30-6283-7820-b73c-d8dc7943bb64\qma-0070-r2` klasöründedir. Ayrıntılı sonuçlar `python-regression/results.json`,
`summary.json`, her scriptin `.log` dosyası ve `go-source-unchanged.json` içindedir.

## Açık kalan gerçek girdiler

QMA-0071: 12 özgün final görsel/metadata, ayrı yetkili AI/admin anahtarları ve
bu içeriklere verilmiş gerçek imzalı onaylar. Kullanıcı bunları son aşamaya bıraktı.

Operasyonel kurulum: bağımsız host/operatör, HTTPS yayını, saat/disk yönetimi,
gerçek scope ve anahtarlar, pinlenmiş oyun policy'si ve gönderim tokenı.
Bu girdiler seçildiğinde servis kurulup kamusal erişim kanıtı ayrıca kaydedilir.
Cüzdan kullanıcı arayüzünde challenge imzalama desteği ayrıca gereklidir.

Gerçek mint, program aktivasyonu, bağımsız servis kurulumu, gerçek anahtar üretimi
veya Git push yapılmadı. r2 ZIP güncel projenin tam yedeği değildir; değişen
kaynakları, kurulum dokümanlarını, önceki dört Python dosyasını ve test kanıtlarını
içeren inceleme/teslim paketidir. Otomatik uygulama betiği içermez.
