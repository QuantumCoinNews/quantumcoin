# QuantumCoin — Aktif QMA takip listesi

**Son güncelleme:** 24 Kasım 2026  
**13 aktif kayıt:** 3 açık · 10 kısmen açık

Bu dosya yalnızca çözülmemiş maddeleri içerir. Her maddede mevcut durum, kalan risk ve kapanış için gereken kanıt yer alır. Testnet ve mainnet aşamalarına ait kararlar birbirinden ayrılmıştır.

**Son doğrulanan kapanış (aktif sayıya dahil değil): QMA-0005.** `go test ./token -count=1` geçti. İlk TOKEN/QC721 tahsisinden sonra mint kapalı; imza alanı network magic, genesis hash ve kalıcı token instance kimliğine bağlı. Token Factory node runtime/consensus özelliği değildir.

## Testnet öncesi — 6 madde

### 55 · QMA-0003 — Kaynak ve runtime ayrımı
**Durum:** Kısmen açık  
**Açıklama:** Ana `C:\Projects\quantumcoin` üzerinde metadata-only envanter 1.575 kaynak yolu gösterdi. Indexte dışlanmış yol yok; mevcut HEAD’de 166 dışlanması gereken tarihsel yol ve çalışma alanında 87 dışlanmış kayıt var: 29 runtime, 21 arşiv, 24 üretilmiş, 8 secret ve 5 karışık custody incelemesi. İçerikler okunmadı; dosya taşınmadı veya silinmedi.  
**Kapanış için:** Runtime/custody verilerini kaybetmeden kaynak ağacından ayırmak, karışık klasörleri incelemek ve yetkili ana tarihçe geçişini tamamlamak. QMA-0224 ve QMA-0250 ile çaprazdır.

### 56 · QMA-0224 — Git geçmişi ve hassas veri temizliği
**Durum:** Kısmen açık  
**Açıklama:** `test_git_privacy` 12/12 geçti; büyük/ikili blob taraması ve sentetik regresyon testi eklendi. r6 steril geçmiş adayı dışarıda üretildi. Ön denetimde 128 commit, 10.692 blob, 6 metin örüntüsü ve 36 ikili/büyük blob bulundu. Adayda 127 commit, 2.426 blob ve 2.391 taranmış metin blobu var; metin bulgusu yok. Kalan 35 ikili/büyük blob ham-byte örüntü taramasından geçti ve eşleşme vermedi. Tek `.exe~` yedek yolu yalnızca aday geçmişten çıkarıldı. Asıl refler değiştirilmedi ve GitHub ana geçmişi yeniden yazılmadı. Aday `e1dac68` kaynak ref anlık görüntüsüne aittir; bu bir yayın veya GitHub geçmiş temizliği değildir.  
**Kapanış için:** 35 ikili varlığın manuel sınıflandırılması, sağlayıcı sırlarının rotasyonu, GitHub cache/fork/remote etkilerinin ele alınması ve yetkili ana tarihçe geçişi. Aday: `%LOCALAPPDATA%/Temp/qc-history-candidate-qma0224-20261124-r6`.

### 57 · QMA-0250 — Çalışma alanı ve geçmiş temizliği
**Durum:** Kısmen açık  
**Açıklama:** Ana çalışma alanı için salt-okunur cleanup planı 170 index/HEAD kaydı buldu: 166 dışlanması gereken tarihsel kayıt (59 arşiv, 21 üretilmiş, 43 manuel inceleme, 42 runtime, 1 secret) ve index ile HEAD’de bulunan 4 release-source-review kaydı. 87 çalışma alanı kaydı ayrı envanterdir. Hiçbir içerik okunmadı; silme veya taşıma yapılmadı.  
**Kapanış için:** Dört release girdisinin kaynak/build/provenance incelemesi, runtime/custody ayrımı ve yalnız doğrulanmış dosyalarda güvenli temizlik. QMA-0003 ve QMA-0224 ile çaprazdır.

### 58 · QMA-0170 — Secret rotasyonu ve Git geçmişi
**Durum:** Açık  
**Açıklama:** Değer göstermeyen tarama `.env` içinde secret biçimli kayıt ve erişilebilir geçmişte eski bir sohbet dışa aktarımında API-key biçimli kayıt buldu. Kullanıcı henüz anahtarları iptal edip yenilemediğini bildirdi. X OAuth 2.0 yenileme kodu hazır; uygulama `client_id`, kullanıcı yetkisi ve token mevcut değil.  
**Kapanış için:** Etkilenen servis anahtarlarını sağlayıcı panellerinde iptal edip yenilemek, yeni değerleri güvenli yerel saklamaya almak ve eski geçmiş/cache etkilerini ayrıca doğrulamak. Değerler rapora yazılmaz.

### 61 · QMA-0206 — Bağımsız yedek ve kurtarma
**Durum:** Kısmen açık  
**Açıklama:** `backuprestore`, `cmd/recovery` ve `policyexamples` testleri geçti; snapshot/rollback, SQLite WAL, bozulmuş yedek reddi ve CLI wallet recovery senaryoları kapsandı. Gerçek yedek oluşturulmadı veya geri yüklenmedi.  
**Kapanış için:** Off-host bağımsız yedek ve ikinci cihazda cüzdan kurtarma tatbikatı; geri yüklenen adresin beklenen adresle eşleştiğinin doğrulanması.

### 63 · QMA-0192 — İmzalı release ve provenance
**Durum:** Kısmen açık  
**Açıklama:** Kilitli Go 1.26.11/MinGW 15.1.0 ortamında release testleri 8/8, signer testleri 3/3 geçti; iki temiz fixture derlemesi eşleşti ve geçici test imzası doğrulandı. Üretim imza anahtarı veya güvenilir public-key kabulü kullanılmadı. İlk testnet sürümünde hardfork aktivasyonu kapalı tutulacak.  
**Kapanış için:** Sahip kontrollü üretim signer’ı, güvenilir anahtarın dağıtımı ve CI’da gerçek imzalı release doğrulaması. Hardfork quorum’u ilk hardfork öncesinde belirlenecek.

## Mainnet öncesi — 4 madde

### 66 · QMA-0071 — Nihai NFT varlıkları ve onayları
**Durum:** Açık  
**Açıklama:** Final artwork ve metadata henüz oluşturulmadı; mevcut plan placeholder niteliğinde. Kullanıcının kararıyla NFT mint kapısı mainnet aşamasına kadar kapalı.  
**Kapanış için:** Nihai varlıklar, metadata/hash doğrulaması ve gerekli AI/admin onayları hazır olmalı; mainnet mint kabulü ayrıca doğrulanmalı.

### 67 · QMA-0107 — BSC ödeme/claim doğrulaması
**Durum:** Açık  
**Açıklama:** Yerel güvenlik/config/precheck testleri geçti. Claim testnet’te kapalı; BSC mainnet kullanımı ve gerçek ödeme doğrulaması son mainnet aşamasına ertelendi. Gerçek contract adresleri ve uçtan uca wallet-signature/claim UI kabulü henüz yok. Gas ücretini kullanıcı BNB ile ödeyecek.  
**Kapanış için:** Mainnet contract ve ağ kimliklerinin doğrulanması, kullanıcı imza akışının kabulü ve gerçek zincirde claim sonucunun kanıtlanması.

### 69 · QMA-0266 — Genesis tahsisi ve rezervler
**Durum:** Kısmen açık  
**Açıklama:** Policy 25,5 milyon QC cap içinde founder payını %5 (1,275 milyon QC: %1 likit, %4 vesting) ve kalan %95’i mining payı olarak tanımlıyor. Ayrı QC bonus/NFT rezervi için genesis tahsisi yok; policy testleri geçti.  
**Kapanış için:** Mainnet founder adresleri, başlangıç tahsisi ve onaylanmış genesis fingerprint’i kesinleştirilmeli. Bunlar olmadan mainnet genesis finalize edilemez.

### 71 · QMA-0272 — Denomination ve ekonomi son kabulü
**Durum:** Kısmen açık  
**Açıklama:** Kanonik ölçek 1 QC = 1.000.000 atom; atom ücret ve UTXO testleri geçti. Kullanıcı mevcut tam QC ödül serisinin korunmasını seçti; kesirli 5,79982325 QC subsidy uygulanmıyor. Fresh V4 ağlarda AtomV3 mikro-fee consensus’i etkin değil.  
**Kapanış için:** Ücret tabanı ve atomik işlem politikasının yetkili hardfork/release kararıyla etkinleştirilmesi ve fresh testnet üzerinde doğrulanması.

## Özellik açılmadan önce — 3 madde

### 72 · QMA-0098 — Canlı sosyal görev doğrulaması ve TGWT dağıtımı
**Durum:** Kısmen açık  
**Açıklama:** Kural ve testler 3.000.000 TGWT toplam havuzunu ve ayrı ayrı doğrulanmış görev başına 1 TGWT’yi koruyor. Telegram, YouTube ve X görevleri testnet’te kapalı; claim mainnet/BSC aşamasına ertelendi. Canlı sağlayıcı yetkilendirmesi ve hesap doğrulaması yapılmadı.  
**Kapanış için:** Her platformun gerçek takip/beğeni kanıtını doğrulayan yetkili entegrasyon, kullanıcı hesabı kabulü ve mainnet claim/BNB gas akışının uçtan uca kanıtı.

### 73 · QMA-0164 — Platform yayın kabulü
**Durum:** Kısmen açık  
**Açıklama:** X için OAuth 2.0 PKCE, CSRF state, loopback callback ve DPAPI saklama/yenileme altyapısı testli; publisher testleri 142/142 geçti. Gerçek X OAuth 2.0 `client_id`/token alınmadı; yayın yolu HTTP 402 billing-required ile duraklatılmış.  
**Kapanış için:** X Developer App ayarı ve hesap yetkisi, uygun API erişimi ve gerçek yayın/teslim doğrulaması. Diğer platformlarda da canlı gönderim kanıtı gerekir.

### 74 · QMA-0165 — Canlı AI içerik kabulü
**Durum:** Kısmen açık  
**Açıklama:** İngilizce içerik, kaynak/iddia kontrolleri, medya üretimi ve FFmpeg teslim hattı fixture’larla testli; publisher testleri 142/142 geçti. Windows DPAPI kasasında kullanılabilir OpenAI anahtarı yok; canlı/ücretli üretim kapalı. Sohbette paylaşılmış eski anahtarın rotasyonu QMA-0170’de açık.  
**Kapanış için:** Döndürülmüş anahtarın güvenli depoya alınması, canlı sağlayıcı üretim/iddia kontrollerinin ve yayın önizlemesinin doğrulanması.