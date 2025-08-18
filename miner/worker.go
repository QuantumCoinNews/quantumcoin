package miner

// Bu dosya ESKİ SÜRÜMLERLE UYUMLULUK için tutulur.
// Asıl işlevler miner.go içinde tanımlıdır:
//
//   - SetGlobalBlockchain
//   - StartMining
//   - StopMining
//   - IsMiningActive
//
// Burada aynı isimleri TEKRAR TANIMLAMAYARAK derleyici çakışmalarını önlüyoruz.
// Eğer geçmişte başka paketler worker.go içindeki bu sembollere referans veriyorsa,
// referanslar miner.go içindeki aynı isimli işlevlere yönlendirilmelidir.
// Gelecekte: çok çekirdekli/iş parçacıklı nonce tarama.
// Şimdilik placeholder; build’i bozmamak için boş bırakıldı.
