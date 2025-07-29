# 💠 QuantumCoin (QC) Whitepaper

## 📌 Amaç ve Vizyon

QuantumCoin, merkeziyetsizlik, güvenlik ve genişletilebilirlik temelinde inşa edilmiş, hem madencilik (PoW) hem de staking (PoS) destekli **hibrit** bir blockchain altyapısıdır. Amacımız, Bitcoin’in sınırlamalarını gidermek, Ethereum’un işlevselliğini yakalamak ve kullanıcı dostu bir deneyimle zincirler arası yenilikleri birleştirmektir.

> "Yapılmayanı yapmak." — QuantumCoin Manifestosu

---

## 🔧 Temel Özellikler

| Özellik                  | Açıklama |
|--------------------------|---------|
| Toplam Arz               | 25.500.000 QC (sabit) |
| Dağıtım                  | %70 Madencilik, %10 Staking, %10 Geliştirici, %5 Yakım, %5 Topluluk |
| Blok Süresi              | ~30 saniye |
| Halving Sistemi          | Her 2 yılda 1 kez |
| Madencilik Yöntemi       | Gelişmiş PoW + GUI destekli |
| Stake Mekanizması        | Cüzdan süresi & bakiyesi temelli |
| NFT Desteği              | QC721 standardı |
| Token Standardı          | QC20 (ERC20 uyumlu) |
| Explorer/API             | Yerleşik HTTP sunucu |
| Çok Dilli Yapı           | EN, TR, ES, ZH |
| Mobil Kazım Desteği      | CPU dostu, platformlar arası kazım |
| Yakım (Burn) Mekanizması | Transferlerde ve blok ödüllerinde aktif |

---

## 🔒 Güvenlik & Doğrulama

- SHA256 & MerkleTree ile blok bütünlüğü
- Transaction signature doğrulama (ECDSA)
- UTXO modeli ile çift harcama engeli
- P2P ağ katmanı: blok & işlem yayını

---

## ⛏️ Madencilik Modeli

- GUI üzerinden başlat/durdur destekli
- Arka planda `miner/worker.go` işleyicisi çalışır
- Performans izleme: `metrics.go`
- Zorluk yönetimi: `difficulty.go`
- Zamanlayıcı destekli kazım: `scheduler.go`
- Bonus/NFT ödülleri: `rewarder.go`, `nft_miner.go`

---

## 💰 Token Ekonomisi

- QC20 standardında token üretimi
- QuantumSwap entegrasyonu (gelecek sürüm)
- Stake havuzu ayrı fon ile yönetilir
- Geliştirici ve topluluk fonları şeffaf biçimde zincirde tutulur

---

## 🎮 Entegrasyonlar ve Gelecek Planı

| Aşama | Hedef |
|-------|-------|
| Q3 2025 | Masaüstü GUI, Mining, Cüzdan, Explorer |
| Q4 2025 | Mobil Kazım, Web Swap, NFT Ödülleri |
| Q1 2026 | Mainnet Yayını, Oyun Entegrasyonu |
| Q2 2026 | DAO Yönetişimi, QuantumBridge, zkSync |

---

## 📚 Teknik Mimariler

**Klasör Yapısı:**

---

## 👥 Topluluk ve Katkı

- Açık kaynak lisansı: MIT
- Katkıda bulunmak için `CONTRIBUTING.md` dosyasına göz atın (yakında)
- Discord, Telegram ve GitHub üzerinden destek alın

---

## ✨ Son Söz

QuantumCoin, teknolojik sınırları zorlayan ve gerçek kullanıcılar için erişilebilir çözümler sunan bir ekosistemdir. Sade değil, güçlü ve genişletilebilir bir sistem kuruyoruz.

> Gelecek zincir üstünde yazılıyor. QuantumCoin ile yazan sen ol.
