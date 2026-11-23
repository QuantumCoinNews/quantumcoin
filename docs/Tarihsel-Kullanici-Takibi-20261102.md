> Kullanıcının gönderdiği tarihsel takip metni; eski işaretler aynen korunur. Güncel NFT durumu `NFT-Calisma-Durumu-20261102.json` dosyasındadır. Başlıktaki 0273, bu dosyanın o maddelere kadar içerik taşıdığı anlamına gelmez.

# QuantumCoin — QMA-0001–QMA-0273 Tam Yapılacak İşler Listesi

Bu dosya altı takip bloğunun tek dosyada birleştirilmiş hâlidir. Maddeler kaynak dosyalardaki sıra ve ifadeler korunarak verilmiştir.


---

# QuantumCoin — Blok 1: QMA-0001–QMA-0050

**Blok durumu:** 🛠 Aktif

## İşaretler

- `[ ]` Bekliyor
- `🛠` Devam ediyor
- `🧪` Test bekliyor
- `✅` Tamamlandı

---

## QMA-0001 özel ilerleme

- 🛠 **QMA-0001 ana maddesi devam ediyor.**
- ✅ `consensus/` klasörü oluşturuldu.
- ✅ `consensus/protocol.go` oluşturuldu.
- ✅ `consensus/protocol_test.go` oluşturuldu.
- ✅ `go test ./consensus` geçti: `0.708s`.
- [ ] `consensus/params.go`
- [ ] `consensus/subsidy.go`
- [ ] `consensus/difficulty.go`
- [ ] `consensus/pow.go`
- [ ] `consensus/transaction.go`
- [ ] `consensus/coinbase.go`
- [ ] `consensus/validation.go`
- [ ] `blockchain/`, `miner/`, `p2p/` entegrasyonları
- [ ] Legacy uyumluluk ve hardfork testleri
- [ ] Tam proje testi

---

- 🛠 **QMA-0001** — **Mimari** — Bağımsız consensus modülü bulunmuyor; blockchain içindeki consensus parçalarıyla çapraz doğrulanacak.
- [ ] **QMA-0002** — **Mimari** — Bağımsız node modülü bulunmuyor; node yaşam döngüsü `main.go`, `cmd`, `internal`, `p2p`, `blockchain`, `miner` ve diğer klasörlere dağılmış olabilir. `node_modules` node modülü değildir.
- [ ] **QMA-0003** — **Yüksek** — Kaynak ağaçta `.exe`, `build/bin`, `bin`, `chain_data.dat`, `miner_address.txt` ve çok sayıda `.bak` derleme/çalışma kalıntısı bulunuyor.
- [ ] **QMA-0004** — **Yüksek** — Mining işlevleri `miner`, `miningtransition`, kök endpoint-lock dosyaları ve `cmd/miner` arasında dağılmış.
- [ ] **QMA-0005** — **Kritik** — Token transfer/yakma/onay işlemlerinin yetkilendirmesi yalnızca metoda verilen adres metinlerine dayanıyor; imza/doğrulanmış çağıran bağlamı çekirdekte yok.
- [ ] **QMA-0006** — **Kritik** — Token, QC20 ve QC721 kritik alanları ve map'leri dışa açık; kilitler ve kontroller doğrudan durum değişikliğiyle aşılabilir.
- [ ] **QMA-0007** — **Yüksek** — QC20 kurucusu aynı arzı hem MaxSupply hem TotalSupply yapıyor; boş owner sahipsiz arz, dolu owner ise baştan dolmuş maksimum arz doğuruyor.
- [ ] **QMA-0008** — **Yüksek** — QC20 eski `Mint` uyumluluk metodu mint hatalarını yutup sessiz başarısızlık üretiyor.
- [ ] **QMA-0009** — **Yüksek** — Registry.Load nil map ve arz/bakiye/owner/symbol bütünlüğünü doğrulamadan disk durumunu yüklüyor.
- [ ] **QMA-0010** — **Kritik** — Miner-token mint köprüsünde block hash/height idempotency, finality ve reorg koruması yok; aynı callback çift mint yapabilir.
- [ ] **QMA-0011** — **Yüksek** — Miner-token köprüsü mint ve Registry.Save hatalarını yutuyor; bellek ve disk durumu ayrışabilir.
- [ ] **QMA-0012** — **Yüksek** — Aynı pakette iki farklı fungible token modeli (`Token`, `QC20Token`) var; davranış ve persistence sözleşmeleri farklı.
- [ ] **QMA-0013** — **Yüksek** — Token adres doğrulaması yalnızca boş olmayan metin kontrolü; format, ağ, checksum ve canonical normalization yok.
- [ ] **QMA-0014** — **Orta** — `StartMiningWithToken` variadic seçeneklerin yalnızca ilkini kullanıp kalanları sessizce yok sayıyor.
- [ ] **QMA-0015** — **Orta** — `AttachToRunningMiner` hiçbir işlem yapmayan dışa açık no-op API.
- [ ] **QMA-0016** — **Kritik** — `api.StartHTTP` varsayılan `:8082` ile tüm arayüzlere açılabilir; env boşken CORS bunu private/dev sayıp wildcard verir.
- [ ] **QMA-0017** — **Kritik** — `resolveHTTPAddr` tam host:port değerlerini başına `:` ekleyerek bozabilir; güvenlik `QC_API_LISTEN` okurken sunucu bu değeri bind adresi olarak kullanmıyor.
- [ ] **QMA-0018** — **Kritik** — Kök HTTPGuard, body limiti, endpoint limiter ve API-token katmanı `api.StartHTTP` zincirine bağlı değil; alternate API yalnızca CORS ile çalışıyor.
- [ ] **QMA-0019** — **Kritik** — Kök `package main` ve `package api` içinde iki farklı API mimarisi var; auth, CORS, route, health ve network-label davranışları ayrışıyor.
- [ ] **QMA-0020** — **Yüksek** — GUI `/api/health` bekliyor; alternate API yalnızca `/health` sunuyor; timeout ve duplicate backend riski var.
- [ ] **QMA-0021** — **Kritik** — GUI `/api/tx/send` için `{from,to,amount,fee,priv_hex}` gönderiyor; endpoint imzalı txDTO beklediği için birinci gönderim yolu sözleşme gereği çalışmıyor.
- [ ] **QMA-0022** — **Kritik** — `/api/send` ikinci ve farklı send sözleşmesi; `priv_hex` yok sayılıyor, yerel CLI auth/method/timeout/concurrency ve güvenli miktar kontrolü olmadan çalıştırılıyor.
- [ ] **QMA-0023** — **Kritik** — Miner start/stop endpointleri method ve API-token zorlamıyor; localhost tarayıcı CSRF'si ve loopback proxy güven sorunu var.
- [ ] **QMA-0024** — **Yüksek** — Miner child process doğal bitince `minerCmd` temizlenmiyor; yeniden başlatma kilitlenebilir. API `run-mine`, teslim edilen miner ise ayrı binary `-address` sözleşmesi kullanıyor.
- [ ] **QMA-0025** — **Kritik** — Blockchain ortak durumu mutex/snapshot olmadan API, miner ve P2P tarafından eşzamanlı okunup yazılıyor; race, tutarsız cevap ve panic riski var.
- [ ] **QMA-0026** — **Yüksek** — Blocks, mempool ve tx status endpointleri pagination/index olmadan tüm veriyi tarıyor; mempool her istekte tüm imzaları tekrar doğruluyor.
- [ ] **QMA-0027** — **Yüksek** — Public CORS bilinmeyen origin'i yalnızca OPTIONS'ta durduruyor; normal GET/POST yan etkili handler'a ulaşabiliyor.
- [ ] **QMA-0028** — **Yüksek** — HTTPGuard ve EndpointLimiter map kayıtlarını hiç silmiyor; IP çeşitliliği bellek büyümesi, global mutex contention ve olay loglama I/O amplification yaratıyor.
- [ ] **QMA-0029** — **Yüksek** — Body limiti alternate API'de etkin değil; limit aşımı standart 413'e çevrilmiyor; MaxHeaderBytes yok.
- [ ] **QMA-0030** — **Yüksek** — `http_server.go.bak_api_route_fix` ve derlenmeyen `api/tx/send` üçüncü send uygulaması kanonik kaynak belirsizliği yaratıyor.
- [ ] **QMA-0031** — **Yüksek** — `/api/block` ve `/api/mine` stub/pasif olmasına rağmen aktif route; yanlış 404/accepted başarı semantiği üretiyor.
- [ ] **QMA-0032** — **Kritik** — Wallet address, network label, AI analysis/bonus ve miner status route sözleşmeleri eksik; SPA catch-all bilinmeyen `/api/*` için HTML 200 döndürebilir.
- [ ] **QMA-0033** — **Orta-Yüksek** — Address endpointleri method ve güvenli adres doğrulaması yapmıyor; blockchain hazır değil durumunu yanlış 400 gösteriyor; boş provider 200 döndürüyor.
- [ ] **QMA-0034** — **Yüksek** — API server graceful shutdown sunmuyor; GUI detached backend ve health uyumsuzluğu orphan/duplicate process ve port çakışması yaratabilir.
- [ ] **QMA-0035** — **Orta-Yüksek** — JSON/düz metin/raw CLI hata ve response şemaları tutarsız; unknown route SPA'ya düşüyor; encode hataları yutuluyor.
- [ ] **QMA-0036** — **Yüksek** — Telemetry, AI alerts, blocks, mempool, balance ve UTXO verileri alternate API'de kimlik doğrulamasız okunabiliyor.
- [ ] **QMA-0037** — **Yüksek** — Kök auth testleri dev modunda tokensız erişimi ve query/form token fallback'ını koruyor; URL/form token sızıntı ve yanlış public-mode halinde auth bypass riski yaratıyor.
- [ ] **QMA-0038** — **Yüksek** — Actual mux, auth entegrasyonu, 413, non-preflight CORS, send/miner methodları, negatif/NaN miktar, timeout, unknown API, SPA fallback, pagination ve race testleri eksik.
- [ ] **QMA-0039** — **Kritik / Mimari** — Projede iki farklı mempool modeli var: `blockchain.Blockchain.pendingTxs` ve `internal.Mempool`. Kabul, doğrulama, kapasite, AI filtresi, çift harcama ve temizleme davranışları farklı; alınan paketlerde `internal.Mempool` çağrısı görünmüyor. İki yol etkinleşirse node'lar farklı işlem havuzları görebilir.
- [ ] **QMA-0040** — **Kritik** — `internal.Mempool.Add` coinbase işlemini açıkça reddetmiyor ve UTXO sahipliği, input miktarı, output miktarı, fee, coinbase maturity ve mempool içi aynı-outpoint çift harcama kontrolü yapmıyor. Yalnız temel imza doğrulaması bu kontrollerin yerini tutmaz.
- [ ] **QMA-0041** — **Kritik / Mimari** — `internal.Mempool` işlem kabulünü node-local `ai.Enabled()` ve `AnalyzeTransactions` sonucuna bağlıyor. AI ayarı/model sonucu node'lar arasında farklı olabileceğinden işlem kabul politikası deterministik değil; analiz ayrıca mempool mutex'i tutulurken çalıştığı için bütün havuzu bloke edebilir.
- [ ] **QMA-0042** — **Yüksek** — `internal.Mempool` transaction pointer'larını doğrudan saklıyor ve `GetTransactions` yalnız slice kopyası döndürüyor. Çağıran taraf kabul edilmiş işlemi sonradan değiştirebilir; ID/index ile içerik ayrışabilir ve yeniden doğrulama olmadan farklı veri kazıma yoluna girebilir.
- [ ] **QMA-0043** — **Kritik** — `internal.UTXOSet.Reindex` harcanmış output'u slice içinden çıkararak kalan output indekslerini sola kaydırıyor. Blockchain outpoint indeksleri sabit olması gerektiğinden sonraki `FindSpendableOutputs` yanlış `OutIndex` döndürebilir. Kanonik `blockchain.UpdateUTXOSet` bu nedenle slotları koruyan farklı bir model kullanıyor.
- [ ] **QMA-0044** — **Yüksek** — `internal.UTXOSet` dışa açık mutable map kullanıyor; mutex, coinbase maturity, nil block/transaction koruması, geçersiz/tekrarlı spend doğrulaması ve zincir snapshot'ı yok. Alınan kapsamda çağrısı görünmüyor; kullanılırsa kanonik UTXO kurallarını atlayabilir.
- [ ] **QMA-0045** — **Kritik / Consensus** — `internal/reward.go` zaman bazlı halving, mining-period cutoff ve sıfıra inen ödül modeli sunuyor; `blockchain` içinde ise yükseklik bazlı kanonik reward/fee/supply consensus'u bulunuyor. Aynı projedeki iki ödül algoritması farklı blok ödülü üretebilir; alınan kapsamda internal reward çağrısı görünmüyor.
- [ ] **QMA-0046** — **Yüksek** — `internal.FeeManager.ApplyFee` verilen transaction'ı tamamen yok sayıp yalnız RAM sayacını artırıyor. Ücretin gerçekten output/input dengesine uygulanması, blok coinbase'ine eklenmesi veya consensus doğrulaması yok; `blockchain` fee consensus'uyla ikinci ve uyumsuz bir model oluşturuyor.
- [ ] **QMA-0047** — **Yüksek / İşlevsel** — `internal.GiveBonus` adı ve miner çıktısı gerçek “100 QC ödülü verildi” anlamı taşıyor fakat kod yalnız konsol/RAM/dosya logu yazıyor; blockchain transaction, balance, mint veya settlement yapmıyor. Kullanıcıya ve diğer modüllere gerçekleşmemiş mali işlem başarılıymış gibi bildiriliyor.
 QMA-0047 — ⚠️ KISMEN ÇÖZÜLDÜ EN SON BAKILACAK.
Sahte 100 QC ödül bildirimi kaldırıldı; ancak gerçek blockchain settlement yapılmadı.
Madde tamamen kapanmadı.
 QMA-0047-S1 — İPTAL EDİLDİ.
Bonus havuzu cüzdanı, özel anahtar güvenliği ve yetkili settlement akışı henüz yapılmadı.
- [ ] **QMA-0048** — **Kritik (etkinleştirilirse)** — Miner yıllık bonus idempotency'si yalnız process içindeki `yearlyGiven` map'inde tutuluyor. Restart sonrası kayıt sıfırlanır; bonus gerçek bakiyeye/NFT'ye bağlandığında aynı dönem için tekrar ödül üretilebilir... TAMAMLANDI...
- [ ] **QMA-0049** — **Yüksek** — Bonus sistemi aynı olayı iki ayrı RAM logunda tutuyor: sınırsız büyüyen `bonusLog []string` ve 1000 kayıtla sınırlı `memBonusLog []Bonus`. Veri modelleri ve saklama sınırları farklı; uzun çalışan node'da bellek büyümesi ve iki API arasında farklı geçmiş riski var. KAOANDI...
- [ ] **QMA-0050** — **Yüksek** — `bonusFilePath` eşzamanlı erişim için korunmuyor; runtime'da `SetBonusFile` ile `GiveBonus/SaveBonus` yarışabilir. Göreli yol `QC_NODE_DIR` yoksa çalışma dizinine düşüyor, parent klasör oluşturulmuyor, rotate/append hataları `GiveBonus` tarafından yutuluyor ve yine de ödül başarılı yazdırılıyor.


---

# QuantumCoin — Blok 2: QMA-0051–QMA-0100

**Blok durumu:** ⬜ Bekliyor

## İşaretler

- `[ ]` Bekliyor
- `🛠` Devam ediyor
- `🧪` Test bekliyor
- `✅` Tamamlandı

--- KAPANDI....

- [ ] **QMA-0051** — **Yüksek / Eksik uygulama** — `persistence` klasörü gerçek persistence katmanı değil, `PLAN_ONLY` veri-domain tanımıdır. DB açma, chain kaydetme/yükleme, transaction, migration, backup, restore, locking ve testnet/mainnet path ayrımı uygulanmıyor; alınan diğer paketlerde bu paketi import eden runtime kodu görünmüyor.
- [ ] **QMA-0052** — **Orta-Yüksek** — Persistence plan doğrulaması yalnız string ve boolean beyanlarını kontrol ediyor. Tanımsız `StorageMode` değerleri kabul edilebilir; gerçek schema/path, append-only davranışı, encryption, backup sonucu, restore testi ve migration uyumluluğu doğrulanmıyor. Testler yalnız plan metadata'sını doğruluyor.
- [ ] **QMA-0053** — **Yüksek / Eksik uygulama** — `queueworker` klasörü gerçek worker/queue sistemi değil, `PLAN_ONLY_NO_WORKER_START` politika listesidir. Enqueue/claim/ack, retry/backoff, lease, idempotency store, dead-letter queue, audit kaydı, admin approval ve shutdown implementasyonu yok; alınan diğer paketlerde runtime bağlantısı görünmüyor.
- [ ] **QMA-0054** — **Orta-Yüksek** — Queue plan validator'ı yalnız `QueueKind` boş mı diye bakıyor ve tanımsız kind değerlerini kabul ediyor. `AutoExecAllowed`, `RequiresAdminApproval` ve queue türü arasında tutarlılık kuralı yok; gerekli queue listesi `ValidatePlanV1` içinde zorunlu değil, ayrı fonksiyona bırakılmış.
- [ ] **QMA-0055** — **Yüksek / Mimari** — `internal` tek pakette bonus, fee, mempool, reward ve UTXO gibi farklı katmanları topluyor; bunların çoğu `blockchain` içindeki kanonik sistemlerle tekrar ediyor. Alınan paketlerde yalnız `miner -> internal.GiveBonus` çağrısı bulundu; diğer parçaların gerçek sahipliği tüm klasörler geldikten sonra kesinleştirilecek.
- [ ] **QMA-0056** — **Yüksek / Test** — `internal` klasöründe mempool, UTXO, reward, fee, bonus concurrency ve restart/idempotency davranışlarını doğrulayan test yok. Persistence ve queueworker testleri çalışıyor fakat yalnız plan sabitlerini/validator'ları test ediyor; gerçek runtime özelliği test edilmiyor.
- [ ] **QMA-0057** — **Yüksek / Mimari** — `burncontrol`, `economycontrol`, `foundervesting`, `hardforkpolicy` ve `networkfee` paketleri çalışma zamanı uygulaması değil, yalnız politika struct'ı ve validator içeriyor. Alınan kaynaklarda bu paketleri import eden runtime kodu yok; kurallar consensus/API/config başlangıcında zorunlu olarak uygulanmıyor.
- [ ] **QMA-0058** — **Kritik / Consensus** — Toplam arz, founder payı, ödül, maturity, burn adresi ve vesting değerleri politika paketleri ile `blockchain` içinde ayrı ayrı sabitlenmiş. Tek kanonik kaynak bulunmadığından bir dosya değiştirilip diğeri unutulursa node'lar belge/politika ile farklı consensus çalıştırabilir.
- [ ] **QMA-0059** — **Yüksek** — Safe-boundary alanları public ve mutable boolean/map beyanlarıdır; davranışsal güvenlik sağlamaz. Validator'lar bazı map anahtarlarını hiç kontrol etmiyor ve gerçek chain/balance/mint işlemlerini doğrulamıyor; “güvenli” sonucu yalnız aynı paket içindeki deklarasyona dayanıyor.
- [ ] **QMA-0060** — **Kritik / Ekonomi** — `networkfee` 0.000001–0.00001 QC mikro ücret tanımlıyor; mevcut blockchain transaction `Amount` ve fee hesabı tam sayı QC kullanıyor. Atom/decimal denomination katmanı olmadan bu ücretler temsil edilemez ve politika uygulanamaz.
- [ ] **QMA-0061** — **Yüksek / Consensus** — Network fee politikası emission sonrası ücretin “node veya block producer”a gideceğini söylüyor; mevcut consensus doğrulanmış fee toplamını coinbase/block producer'a ekliyor. Sıradan node operatorüne ödeme, hak sahipliği ve deterministik seçim mekanizması yok.
- [ ] **QMA-0062** — **Kritik / Ekonomi** — Burn sistemi yalnız `QC_BURN_SINK` takma adı ve boolean şartları tanımlıyor. Adresin gerçekten geçersiz/harcanamaz olduğuna dair kriptografik script/anahtar kanıtı, burn transaction sınıflandırması ve circulating-supply muhasebesi yok; mevcut wallet adres formatıyla uyumu da doğrulanmıyor.
- [ ] **QMA-0063** — **Yüksek** — `economycontrol.ValidatePolicyV1` allocation isimlerini ve toplam 10000 basis point'i kontrol ediyor fakat her ismin doğru paya ve doğru Locked/Liquid semantiğine sahip olduğunu doğrulamıyor. Founder ve network allocation değerleri yer değiştirilerek veya flag'ler bozularak validator geçebilir; blockchain'deki bonus/NFT/mining alt havuzları da bu policy içinde zorunlu değil.
- [ ] **QMA-0064** — **Kritik / Consensus** — Founder vesting gerçek consensus'ta founder adreslerini `config.Current()` ve `QC_MAIN_ADDRESS` ortam değişkeninden çalışma anında okuyor. Adresler genesis/chain parametresine kriptografik olarak bağlanmadığından node config farkı vesting'in farklı adrese uygulanmasına veya tamamen devre dışı kalmasına yol açabilir; policy'nin “mainnet sonrası değişmez” şartı runtime tarafından garanti edilmiyor.
- [ ] **QMA-0065** — **Yüksek / Yönetişim** — Hardfork policy yalnız boolean onaylar ve sebep listesi taşır. Sebeplerin exact allowlist olduğu, tekrar/çelişki bulunmadığı, founder/admin onaylarının imzalı-quorumlu olduğu, snapshot/binary hashlerinin gerçekten doğrulandığı ve migration height'ın consensus'a işlendiği bir mekanizma yok.
- [ ] **QMA-0066** — **Kritik / Mimari** — NFT için üç ayrı ve bağlantısız model var: `nft` placeholder/seçim paketi, `blockchain.MintNFT` metadata yazımı ve `miner.GrantNFTBonus`. Alınan kapsamda `quantumcoin/nft` runtime tarafından kullanılmıyor; aynı “NFT verildi” olayı farklı sistemlerde farklı anlam taşıyor.
- [ ] **QMA-0067** — **Yüksek / İşlevsel** — `nft.CheckAndMintMinerNFT` dışa açık bir stub olup her çağrıda boş NFT kimliği ve `nil` hata döndürüyor. Çağıran taraf mint başarılı sanabilir; alınan kaynaklarda çağrısı henüz görünmüyor.KAPALI.
- [ ] **QMA-0068** — **Kritik / Blockchain** — `blockchain.MintNFT` NFT transaction/ownership kaydı üretmek yerine son bloğun `Metadata` map'ini sonradan değiştiriyor. Metadata canonical PoW hashine dahil değil; kayıt consensus ile korunmuyor, node'lar arasında farklılaşabilir ve sahiplik/transfer/tekillik kanıtı oluşturmaz. NFT ID'si ayrıca local `time.Now().UnixNano()` ile üretildiği için deterministik değildir.
- [ ] **QMA-0069** — **Kritik / İşlevsel** — `miner.GrantNFTBonus` chain mint başarısız olsa bile DB hook ve Telegram bildirimine devam ediyor; iki ayrı env flag nedeniyle miner reward açıkken chain mint kapalı olabilir. Boş txID ile ödül kaydı/bildirimi oluşarak kullanıcıya verilmemiş NFT verilmiş gibi gösterilebilir. YAPILDI.
- [ ] **QMA-0070** — **Yüksek / Adalet-Güvenlik** — Aylık NFT aday değerlendirmesi her boş olmayan adresi `Eligible=true` yapıyor; oyun faaliyeti, adres formatı, tek kullanıcı/tek cüzdan ve dönem kanıtı yok. Kazanan seçimi public/default seed, aday listesinin tamamı ve hash'in yalnız ilk byte'ı üzerinden modulo kullanıyor; duplicate/Sybil adaylar ve modulo bias seçimi etkileyebilir. AÇIK
- [ ] **QMA-0071** — **Yüksek / İşlevsel** — `ValidateMonthlyNFTMainnetMintGateV1` approval tokenı kabul etse bile ardından `ValidateMonthlyNFTPlanV1` her `MintEnabled` veya `Minted` değeri için hata veriyor. Bu nedenle gate gerçek mint-ready plana geçiş sağlayamıyor; final artwork/metadata hashleri, uniqueness ve AI/admin kanıtları da kontrol edilmiyor. AÇIK
- [ ] **QMA-0072** — **Yüksek / Veri bütünlüğü** — Kodla üretilen 2026 aylık NFT planı ile checked-in JSON plan/stock eşleşmiyor: kod Ocak'a “Lunar Pioneer” atarken JSON/stock bunu Haziran'a atıyor. Stock 12 metadata/image yolu gösterse de pakette yalnız Haziran metadata'sı var; testler dosya varlığını ve kod-JSON eşitliğini kontrol etmiyor.
- [ ] **QMA-0073** — **Yüksek / Test** — Altı paketin testleri geçiyor ancak çoğunlukla default policy ve birkaç boolean değişimini sınar. Runtime entegrasyonu, tek kanonik ekonomi kaynağı, mikro fee temsili, founder config node ayrışması, gerçek burn, hardfork imza/quorum, NFT mint failure, catalog drift ve consensus persistence senaryoları kapsanmıyor.