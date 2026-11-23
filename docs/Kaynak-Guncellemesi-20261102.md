# QMA-0001–0073 kaynak güncellemesi

2 Kasım 2026. Bu teslim, 0001'den mevcut 0073 grubuna kadar yerelde geliştirilmiş
kaynak kodu, test ve dokümanları deponun ana klasör yapısına taşır. Sonraki
maddeler bu teslim kapsamında değildir. Her maddenin kapandığı iddiası taşımaz;
0001 özel ilerleme, 0071 final içerik/onay bekleme durumu korunur.

Kullanıcının verdiği takip metni `Tarihsel-Kullanici-Takibi-20261102.md`
dosyasına tarihsel kaynak olarak alınmıştır. Metindeki eski sorun açıklamaları
ve işaretler değiştirilmemiştir. Son NFT durumu için
`NFT-Calisma-Durumu-20261102.json` ve
`NFT-Secim-ve-Mint-Teknik-Sonucu-20261102-r2.md` esas alınır.

## Aktarılan teknik alanlar

- Consensus, imzalı protokol/aktivasyon, chain doğrulama ve kalıcılık.
- Token yetkileri, idempotency, adres ve registry bütünlüğü.
- API auth/CORS, miktar/ücret sözleşmeleri, miner ve cüzdan bağlantıları.
- Kalıcı queue/worker, restart, fee/burn/vesting ve ekonomi kuralları.
- Kanonik NFT yaşam döngüsü, doğrulanmış Telegram faaliyeti ve cüzdan kanıtı,
  sabit dış beacon, kuruluma hazır bağımsız tanık ve final onay kapısı.
- Bu alanların kaynak testleri ve açıklayıcı dokümanları.

GitHub `main` üzerindeki bağımsız autopublisher değişiklikleri ve mevcut diğer
dosyalar temel alınmıştır. Yerel kullanıcı index'i ve çalışma dalı değiştirilmeden
ayrı bir Git index'i ile yayın ağacı hazırlanır. İki eski/çakışan kaynak girişi
(`api/tx/send`, `blockchain/blockchain/tx.go`) kaldırılır. Önceden takip edilen
`.wallet` dosyası güncel ağaçtan çıkarılır; yerel kopyası gönderilmez. Bu işlem
geçmiş commitleri silmez; eski gerçek anahtarlar için anahtarın yenilenmesi ayrıca
değerlendirilmelidir.

Çalışma zamanı DB/log, anahtar, yedek dosyaları ve medya çıktıları bu kaynak
teslimine eklenmez. Canlı mint/aktivasyon, yeni binary release veya sunucu
kurulumu yapılmaz. QMA-0070 kuruluma hazır teknik kapsamda tamamdır;
QMA-0071 gerçek 12 varlık ve yetkili AI/admin onayları için açıktır.

Önceki doğrulama: tam Go/race koşusunda 1.088 ana test ve 24 paket başarılı;
son NFT regresyonunda 23 script, 98 unittest vakası ve 13 assertion scripti
başarılıdır. GitHub yayını için hazırlanan kaynak ağacı ayrıca doğrulanır ve
yayın commit'i teslim raporunda kaydedilir.
