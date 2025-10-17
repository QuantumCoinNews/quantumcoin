// === Timing ===
export const BLOCK_TIME_SECONDS = 30;   // Oyundaki tur süresi (her 30 sn bir blok üretimi)
export const MINING_SECONDS = 6;    // Gemi bir bölgede kazarken gösterilecek animasyon süresi
export const BATCH_WRITE_SECONDS = 180;  // Sunucunun pending bakiyeleri zincire toplu yazma aralığı (3 dk)

// === Reward Schedule ===
export const BLOCK_REWARD_QC = 50;   // İlk 2 yıl için sabit blok ödülü
export const PHASE_YEARS = 2;    // İlk faz süresi (yıl)
export const USE_DYNAMIC_DIFFICULTY = false;// Ağ yüküne göre dinamik katsayı (şimdilik kapalı)
export const DIFFICULTY_MIN = 0.25; // (aktif edilirse) ödül alt sınırı
export const DIFFICULTY_MAX = 1.00; // (aktif edilirse) ödül üst sınırı
export const CONFIRMATION_REQUIRED = 1;    // Confirmed sayılması için gereken minimum onay (tx inclusion)

// === Zones (rota) ===
export const MINING_ZONES = [
    { id: "zone-1", pos: { x: -12, y: 0, z: -10 } },
    { id: "zone-2", pos: { x: 8, y: 2, z: -14 } },
    { id: "zone-3", pos: { x: 14, y: -1, z: -6 } },
    { id: "zone-4", pos: { x: -6, y: 1, z: -4 } },
    { id: "zone-5", pos: { x: 2, y: -2, z: -12 } }
];

// === Energy (otomatik modda pasif) ===
export const MINING_ENERGY_COST = 0;
export const MAX_ENERGY = 100;
export const ENERGY_REGEN_PER_MIN = 5;
export const COOLDOWN_SECONDS = 0;

// === Storage Keys ===
export const STORAGE_KEYS = {
    ENERGY: "qc_energy",
    LAST_ENERGY_AT: "qc_last_energy_at",
    QC_BALANCE: "qc_balance",
    ADDRESS: "qc_address",            // Kullanıcı cüzdan adresi
    LAST_BLOCK_AT: "qc_last_block_at" // Son hesaplanan blok zamanı (epoch seconds)
};
