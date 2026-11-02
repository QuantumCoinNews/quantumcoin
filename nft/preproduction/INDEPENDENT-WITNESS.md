# Bağımsız dış taahhüt tanığı — kurulum sözleşmesi

Uygulama: `telegram_game/telegram_game/nft_witness_server_v1.py`.
Bu bileşen oyun/node'dan **bağımsız operatör ve sunucuda** çalıştırılmak içindir.
Oyun sunucusunda aynı yetkilinin yönettiği bir kopya, bağımsız dış kanıt sayılmaz.
Bu kaynak teslimi gerçek bir bağımsız sunucuya kurulmuş olma iddiası taşımaz.

## Operatör girdileri

- `QMA_WITNESS_KEY_FILE`: operatörün kendi Ed25519 anahtarının bulunduğu mutlak
  dosya yolu; 32 bayt anahtarın küçük harf hex gösterimi. Agent gerçek anahtar üretmez.
- `QMA_WITNESS_DB_FILE`: ayrı kalıcı SQLite DB'sinin mutlak yolu.
- `QMA_WITNESS_SCOPE_FILE`: izin verilen network/campaign/genesis JSON'unun
  mutlak yolu. Anahtarlar `networkID`, `campaignID`, `legacyGenesisHex`.
- `QMA_WITNESS_PUBLIC_ORIGIN`: herkesin erişebildiği HTTPS origin'i.
- `QMA_WITNESS_SUBMIT_TOKEN`: yalnız oyun servisinin bildiği, 32–256 görünür ASCII
  karakterden oluşan ayrı gönderim kimlik bilgisi. QC/node API anahtarı kullanılmaz.

Bağımsız hostta ayrı sanal ortama
`python -m pip install -r telegram_game/requirements-nft-witness.txt`
ile servis bağımlılıkları kurulduktan sonra, operatörün
servis yöneticisi şu uygulamayı HTTPS reverse proxy arkasında başlatır:

```text
uvicorn telegram_game.nft_witness_server_v1:create_app --factory --host 127.0.0.1 --port 8097
```

Komut `telegram_game/` paket kökünde veya o kökü içeren PYTHONPATH ile çalışır.
Saat senkronizasyonu, kalıcı disk, anahtar erişim izinleri ve public HTTPS
yayını operatörün sorumluluğundadır. Gerçek scope ve anahtarlar bu teslimde seçilmedi.

Scope dosyası gerçek doğrulanmış programdan doldurulur; aşağıdaki boş şablon
çalışan scope değildir:

```json
{"networkID":"", "campaignID":"", "legacyGenesisHex":""}
```

Oyun policy'sindeki `publicationWitness.publicKey`, bu bağımsız operatörün
Ed25519 açık anahtarının 32 bayt hex gösterimidir. `urlPrefix` tam olarak
`https://OPERATÖR/entries/`, `witnessSubmitURL` ise
`https://OPERATÖR/v1/commitments` biçimindedir. Gerçek host adı kullanılmalıdır.

## Doğrulanabilir davranış

`POST /v1/commitments`, `X-QMA-Witness-Token` başlığı ve `{"commitment": ...}`
gövdesi alır. Yalnız program/dönem/havuz hash'i/adet/policy/beacon taahhüdü
gönderilir; oyuncu listesi, özel anahtar ve QC yetki ayarı gönderilmez.

Network + campaign + genesis + dönem için **ilk kayıt** kalıcıdır. Aynı kayıt
tekrar okunabilir; başka havuz/policy/round ile değiştirmek 409 döndürür. Dönem
başına tek scope kimliği, aynı kullanıcı grubuna birden fazla taahhüt yayımlayıp
beacon sonrasında avantajlı olanı seçme yolunu kapatır. Yerel oyun DB'si bu
dış kaydı başka bir sonuçla değiştiremez.

`GET /entries/{scopeHash}` kimlik doğrulama istemeyen kamusal, imzalı kaydı
döndürür. Scope hash'i `QC_NFT_PUBLICATION_SCOPE_V1\0` alan ayrımı ve sıralı
ASCII kanonik JSON ile SHA-256'dır. Kayıt URL'si istek Host başlığından türetilmez;
operatörün sabit HTTPS origin'inden türetilir.

Oyun, gönderim yanıtı kaybolsa da **aynı scope URL'sini** sorgular. İmza/hash/
program/dönem/zaman kontrolü ve kamusal GET tamamlanmadan yerel kayıt yetki
almaz. Son tarih geçtiyse farklı tur veya yeni havuza otomatik geçilmez.

## Oyun tarafı

Yerel `QC_NFT_EVIDENCE_POLICY_PATH` JSON'unda witness açık anahtarı, HTTPS
`urlPrefix`, `externalServices.witnessSubmitURL` ve sabit drand relay origin'i
pinlenir. `QC_NFT_WITNESS_SUBMIT_TOKEN` yalnız bağımsız tanığa POST için kullanılır.
Beacon ve kamusal kayıt GET'lerine kimlik bilgisi gönderilmez; redirect izlenmez.

Yalnız mevcut admin kimlik doğrulamasıyla:

1. `/api/admin/nft/monthly/publish-draw-commitment`: kalıcı outbox → dış ilk kayıt →
   kamusal GET ve imza doğrulama → yerel atomik kayıt.
2. `/api/admin/nft/monthly/draw-from-beacon`: pinlenmiş `/chainHash/info` eşleşmesi →
   yalnız sabit `/chainHash/public/round` → gerçek BLS doğrulaması → atomik çekiliş.

Quicknet `bls-unchained-g1-rfc9380` ve `pedersen-bls-unchained` ayrı imza/key
boyutlarıyla desteklenir. Eski hatalı domain kullanan `bls-unchained-on-g1`,
`latest`, farklı round ve imzasız randomness kabul edilmez.

Bu dosya canlı kurulum komutu çalıştırmaz; gerçek operatör/sunucu tanımlanınca
kurulum ve dış erişim kanıtı ayrıca kaydedilmelidir.
