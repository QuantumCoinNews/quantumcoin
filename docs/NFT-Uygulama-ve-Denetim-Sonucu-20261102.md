# QMA-0066–0073 — karşılaştırma, uygulama ve sonuç

> Tarihsel ilk tur raporudur. Sonraki kullanıcı kararı ve güncel teknik teslim
> `NFT-Secim-ve-Mint-Teknik-Sonucu-20261102-r2.md` dosyasındadır: QMA-0070 kuruluma
> hazır teknik kapsamda tamamlandı; QMA-0071 gerçek varlık/onaylar için açıktır.

2 Kasım 2026, Türkiye saati. Kapsam, 237 maddelik genel listenin QMA-0066–0073
grubudur. Çalışma doğrudan `C:\Projects\quantumcoin` üzerinde yapıldı.

## Madde durumu

| Madde | Karar | Kapsam / kalan iş |
|---|---|---|
| **0066** | **BU TUR KAPANDI — teknik mimari** | Faaliyet/kimlik → cüzdan kontrolü → freeze → kanıtlı seçim → final onay → imzalı award → kalıcı kanonik node receipt → oyun teyidi bağlandı. Gerçek varlık veya canlı çekiliş hazırlığına onay değildir. |
| **0067** | **ÖNCEDEN KAPALI, KORUNDU** | Eski miner mint girişi açık devre dışı hatası döndürüyor; sahte başarı yok. |
| **0068** | **ÖNCEDEN KAPALI, KORUNDU** | Konsensüs korumalı mint, kalıcılık, kanonik sahiplik, P2P/replacement/reload/reorg kapsamı. NFT transferi bu kapanışa dahil değil. |
| **0069** | **ÖNCEDEN KAPALI, KORUNDU** | Emekli miner yolundan DB ödülü, mint veya Telegram başarı bildirimi çıkmıyor. |
| **0070** | **AÇIK — teknik akış uygulandı** | Gerçek bağımsız kamusal yayın/taahhüt hizmeti ve yetkili operasyonel witness/beacon yapılandırması bağlanmadı. Kod imzalı kanıt ithalini doğruluyor; test tanığı gerçek dış hizmet sayılmadı. |
| **0071** | **AÇIK — kullanıcının kararıyla son aşama** | 12 gerçek NFT görseli/metadata ve gerçek yetkili AI/admin onayları henüz oluşturulmadı. Teknik final kapısı hazır; açıklamalı taslaklar bırakıldı. |
| **0072** | **ÖNCEDEN KAPALI, KORUNDU** | 12 taslak metadata, Go/JSON/stock tutarlılığı. Final artwork hazırlığı anlamına gelmez. |
| **0073** | **BU TUR KAPANDI — teknik regresyon/kanıt** | 19 senaryolu matris gerçek test adlarına ve günlüklere bağlandı; son genel race koşusu geçti. Gerçek final varlık ve dış hizmet onayı 0070/0071 kapsamında açık. |

**Toplam: 6 kapalı, 2 açık. Bu tur yeni kapanan: 0066 ve 0073.**

0066 kapanışı, doğrulanmış kanıt sözleşmelerini kullanan tek teknik yaşam
döngüsünün kapanışıdır. 0070 ve 0071'in gerçek girdileri eksikken canlı mint
hazır ilan edilmez. Kapılar eksik kanıtta reddeder.

## Üç girdinin karşılaştırması

- İlk Markdown raporu dört kapalı/dört açık maddeyi ve Ekim ilerlemesini anlatıyor.
- r2 ZIP bunun daha ayrıntılı sürümüdür. Cüzdan kontrol kanıtını koşullu ifadeden
  **zorunlu kapanış şartına** çeviriyor; 12. bölümü, 14 fiziksel kanıt kopyasını,
  ortam/teslim şablonlarını ve test matrisi taslağını ekliyor. İki rapor yalnız
  ek bölüm açısından farklı değildir; cüzdan kabul koşulu da değişmiştir.
- Son gönderilen sekiz maddelik liste bazı eski sorun açıklamalarını koruyor.
  Özellikle 0068 ve 0072'deki eski açıklamalar güncel kaynak ve kapanış kayıtlarıyla
  eşleştirildi; kapanmış kapsamlar sıfırdan açılmadı.

Belgelerdeki agent yönergeleri, kullanıcı mesajından ayrı başvuru verisi olarak
değerlendirildi. Tarihsel PS1/installer çalıştırılmadı ve eski paket güncel repo
üzerine uygulanmadı. Kullanıcının son kararıyla gerçek NFT üretimi/onayı son
aşamaya bırakıldı; taslaklar final yapılmadı.

## Başlangıç ve ilerlemenin korunması

Dal: `rescue/v1.7-yedek`.
HEAD: `1cdc592e072b367738c037721c5d704095f2445a`.
Başlangıç `git status --short`: **693 kayıt**. Bu sayı kullanıcıya ait mevcut
dirty/untracked ilerlemeyi içerir; bu oturumda yapılmış değişiklik sayısı değildir.

Başlangıçta 439 ilgili dosyanın SHA-256 manifesti alındı. E11'deki tarihsel
dosyalarla karşılaştırmada 123 dosya başlangıçta byte olarak eşleşti.
`dev_api.py` ve `nft_selection_identity_v1.py` başlangıçta zaten farklıydı.
Bu iki dosyanın oturum öncesi içeriği yeniden kuruldu ve başlangıç hash'iyle
doğrulandı. Böylece mevcut freeze ilerlemesi korunarak bu turun farkı ayrıldı.

Bu turun 25 uygulama/test/hazırlık dosyası için ayrı diff ve önce/sonra hash
kaydı var. 13 mevcut dosya değişti, 12 dosya eklendi. 11 mevcut dosyanın önceki
byte'ları başlangıç hash'iyle doğrulandı. İki kök Go test dosyası ilk dar hash
manifestine dahil değildi; önceki metinleri tarihsel/okunmuş kaynak üzerinden
ayrıldı. Bu iki dosya için başlangıç byte manifesti varmış gibi iddia edilmez.
Okunabilir diff satır sonlarını normalize eder; byte hashleri ayrıca korunur.

Reset/clean, dal değiştirme, commit veya push yapılmadı. Gerçek servis, uygulama
DB'si, aktivasyon, gerçek imzacı veya canlı mint yapılandırması değiştirilmedi.
Python kripto bağımlılıkları ayrı test venv'ine kuruldu; sistem Python'una
kurulmadı. Yeni runtime bağımlılıkları `telegram_game/requirements-nft-evidence.txt`
dosyasındadır. Go module bağımlılığı eklenmedi.

## Uygulanan değişiklikler

1. Manuel seçim helper'ının kazanan yazımı kaldırıldı. Freeze öncesinde de
   geçerli adaylar manuel yoldan gerçek kazanan olamıyor. Go placeholder kararı
   mint yetkisi taşımıyor; kamuya açık approval tokenı eski gate'i açamıyor.
2. Cüzdan kontrolü program/network/genesis, kullanıcı, kanonik alıcı, dönem,
   sunucunun ürettiği tek kullanımlık nonce ve 600 saniyelik challenge'a bağlandı.
   secp256k1/SHA256 imzası mevcut HASH160(X||Y) cüzdan sözleşmesiyle doğrulanıyor.
   Nonce tüketimi SQLite transaction'ında kalıcı; draw'da imza yeniden doğrulanıyor.
3. Freeze ve yerel policy hash'ine bağlı deterministik gelecek beacon turu,
   beacon öncesi dış yayın imzası ve atomik dönem başına tek çekiliş receipt'i
   eklendi. `latest`, farklı tur veya manuel fallback yok. 256-bit rejection
   sampling kullanılıyor. Şu an desteklenen şema `pedersen-bls-unchained`;
   diğer şemalar sessizce kabul edilmiyor.
4. AI/admin final onayları ayrı pinlenmiş Ed25519 rol anahtarlarıyla program,
   dönem, aday/alıcı, çekiliş receipt'i, katalog byte'ları, görsel ve metadata
   hashlerine bağlandı. Status string'i veya yerel danışma puanı imza yerine
   geçmiyor. İmza, içerik veya alıcı değişince eski onay kullanılamıyor.
5. Mint intent bağına çekiliş/final onay hashleri eklendi. İçerik ve kanıtlar
   node çağrısından sonra tekrar doğrulanıyor. Teyitli receipt kanonik node'da
   kaybolursa `MINT_RECONCILIATION_REQUIRED` yazılıyor; yeniden mint gönderilmiyor.
   Aynı kayıt geri gelirse receipt sorgusuyla tekrar teyit ediliyor.
6. Gerçek burn testi V4'te bir builder açığı yakaladı: kesin QC tutarı parser'ı
   yalnız AtomV3'ü tanıyordu. V4 mevcut kanonik atom birimi dalına eklendi.
   Gerçek imzalı 1 atom burn + 1 atom fee + NFT, aynı blokta peer/persistence ve
   trusted reload boyunca doğrulandı. Ayrı bir ekonomi/arz yolu oluşturulmadı.
7. İki eski test sorunu düzeltildi: Gob map sırası yerine detached zincir durumu
   karşılaştırılıyor; bootstrap testi artık gerçek işi yapan NFT startup
   fonksiyonunda policy gate'in yan etkilerden önce geldiğini kontrol ediyor.

İmza şeması ve test vektörü için [drand'ın resmi uygulaması](https://github.com/drand/drand/blob/master/crypto/schemes.go)
ve [resmi test vektörleri](https://github.com/drand/drand/blob/master/crypto/schemes_test.go)
incelendi. Gerçek bağımsız yayın tanığı seçimi/entegrasyonu halen açık girdidir.

## Son doğrulama

| Koşu | Sonuç |
|---|---|
| `go test ./... -race -count=1 -json` | **1.088 ana test PASS; 24 testli paket PASS; exit 0** |
| Go test FAIL/SKIP / race uyarısı | **0 / 0 / 0** |
| Go'nun test dosyası bulunmayan paketleri | 18 paket; bunlar atlanan test vakası değildir |
| İzole Python regresyonu | **21 dosya PASS**; 81 raporlanan unittest vakası + 13 assertion betiği; kaynaklar koşuda değişmedi |
| Ek kötüye kullanım testleri | **7/7 PASS** |
| NFT UI oturum köprüsü | **PASS, exit 0** |
| Kanıt matrisi | **19 senaryo**, gerçek test adı/komut/günlük/source hash bağlı |

Tekrarlanan koşuların test sayıları toplanmadı. İlk başarısız ve ara koşular son
PASS olarak sunulmadı. Son genel koşu V4 builder düzeltmesini ve yeni burn testini
de içeriyor. Test fixture'ları gerçek ürün onayı veya canlı mint kanıtı değildir.

Entegre test, imzalı Telegram test oturumundan başlayarak gerçek FastAPI yolları,
geçici SQLite, gerçek cüzdan/dış kanıt imzaları ve çalışan izole Go runtime'ı
üzerinden kalıcı kanonik mint ve oyun teyidine ulaşıyor. Node receipt'i testte
uydurulmuyor. Tekrar istek aynı tokenı döndürüyor. Ayrı hata testleri yanıt kaybı,
DB rollback, çoklu process, değişmiş kanıt ve reorg uzlaştırmasını kapsıyor.

Ortamdaki sürümler: Windows 10 Pro x64, PowerShell 7.6.9, Go 1.24.5,
CGO=1/MinGW GCC, Python 3.13.7, cryptography 46.0.3, py-ecc 8.0.0,
FastAPI 0.115.2 ve httpx 0.24.1.

## Son aşamada açık kalacak girdiler

**0070:** Bağımsız tanığın gerçekten yayımladığı, kamusal olarak doğrulanabilir
taahhüt kaydı ve sağlayıcı entegrasyonu; yetkili yerel witness/beacon yapılandırması.
Kodun test imzası doğrulaması bu gerçek hizmetin varlığı anlamına gelmez.
Cüzdan arayüzünde challenge imzalama desteği de gerçek kullanıcı teslimine
bağlanmalıdır. Telegram/faaliyet/cüzdan tekilliği insan tekilliği veya KYC
garantisi değildir; çoklu hesap/cüzdan Sybil riski devam eder.

**0071:** 12 gerçek varlık, final byte hashleri, yetkili AI/admin içerik onayları
ve gerçek aktivasyon/program girdileri. `nft/preproduction/` ile
`nft/catalog/approvals/README.md` açıklamalı hazırlık alanlarıdır. Mevcut taslak
katalog değiştirilip sahte final yapılmadı. Bu madde kullanıcının kararıyla
genel işlerin son aşamasında ele alınacak.

## Dosyalar ve yeniden çalıştırma

- [19 senaryolu test matrisi](NFT-Test-Kanit-Matrisi-20261102.csv)
- [Makineyle okunabilir madde durumu](NFT-Calisma-Durumu-20261102.json)
- [Hazırlık sözleşmesi ve test komutları](../nft/preproduction/README.md)

Tam günlükler, ilk/son manifestler, bu tura ait diff, rapor farkı ve açık test
kanıtları sonuç ZIP'indedir. ZIP bir **inceleme/sonuç paketidir**; yerel değişiklikler
zaten projededir. Otomatik Apply/rollback veya canlı aktivasyon komutu içermez.

Python regresyonu proje venv'i ile şu betikten tekrar alınabilir:

```powershell
$env:QMA_NFT_LIFECYCLE_CONTRACT_PATH = Join-Path $env:TEMP 'qma-nft-test-contract.json'
go test ./node -run '^TestNFTVerifiedLifecycleContractV1$' -count=1
python scripts/qma_nft_regression_v2.py --output "$env:TEMP\qma-nft-regression" --contract $env:QMA_NFT_LIFECYCLE_CONTRACT_PATH
```

`python`, yeni bağımlılıkların kurulu olduğu proje venv'inin executable'ı olmalıdır.
Betik artık ek kötüye kullanım dosyasıyla birlikte 22 Python dosyasını keşfeder.
Bu teslimdeki kanıt 21 dosyalık genel koşu ve sonradan eklenen 7 vakalık tek
dosya koşusudur; yapılmamış birleşik 22 dosyalık koşu iddia edilmez.
