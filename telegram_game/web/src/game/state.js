import {
    MINING_ENERGY_COST,
    MAX_ENERGY,
    ENERGY_REGEN_PER_MIN,
    STORAGE_KEYS
} from "./constants";

/**
 * Basit kalıcı durum: localStorage varsa kullanır, yoksa RAM'de tutar.
 */
const hasLocalStorage = (() => {
    try {
        const k = "__qc_test__";
        window.localStorage.setItem(k, "1");
        window.localStorage.removeItem(k);
        return true;
    } catch {
        return false;
    }
})();

const memoryStore = new Map();

function readStore(key, fallback = null) {
    if (hasLocalStorage) {
        const raw = window.localStorage.getItem(key);
        return raw === null ? fallback : JSON.parse(raw);
    }
    return memoryStore.has(key) ? memoryStore.get(key) : fallback;
}

function writeStore(key, value) {
    if (hasLocalStorage) {
        window.localStorage.setItem(key, JSON.stringify(value));
    } else {
        memoryStore.set(key, value);
    }
}

const nowSec = () => Math.floor(Date.now() / 1000);

/**
 * Enerji durumunu yükle / başlat.
 * ENERGY: mevcut enerji (0..MAX_ENERGY)
 * LAST_ENERGY_AT: en son enerji güncelleme zamanı (epoch seconds)
 * QC_BALANCE: gösterim için (opsiyonel, Mining.jsx de yönetebilir)
 */
export function loadEnergyState() {
    const energy = readStore(STORAGE_KEYS.ENERGY, MAX_ENERGY);
    const lastAt = readStore(STORAGE_KEYS.LAST_ENERGY_AT, nowSec());
    return { energy, lastAt };
}

export function saveEnergyState({ energy, lastAt }) {
    writeStore(STORAGE_KEYS.ENERGY, clamp(energy, 0, MAX_ENERGY));
    writeStore(STORAGE_KEYS.LAST_ENERGY_AT, lastAt ?? nowSec());
}

export function clamp(v, a, b) {
    return Math.max(a, Math.min(b, v));
}

/**
 * Aradan geçen süreye göre enerji yeniler.
 * Dakikada ENERGY_REGEN_PER_MIN kadar enerji dolar (tamsayı).
 * Geriye güncellenmiş {energy,lastAt} döner.
 */
export function regenEnergy(state) {
    const tNow = nowSec();
    const dt = Math.max(0, tNow - state.lastAt);
    if (dt < 60 || state.energy >= MAX_ENERGY) {
        // 1 dakikadan az geçtiyse veya zaten doluysa küçük optimizasyon
        return state;
    }

    const minutes = Math.floor(dt / 60);
    const gain = minutes * ENERGY_REGEN_PER_MIN;

    const nextEnergy = clamp(state.energy + gain, 0, MAX_ENERGY);
    const consumedSeconds = minutes * 60;
    const nextLastAt = state.lastAt + consumedSeconds;

    const updated = { energy: nextEnergy, lastAt: nextLastAt };
    saveEnergyState(updated);
    return updated;
}

/**
 * Enerji harcar; yeterli değilse false döner.
 */
export function trySpendEnergy(state, cost = MINING_ENERGY_COST) {
    // harcamadan önce regen uygula
    const s = regenEnergy(state);
    if (s.energy < cost) return { ok: false, state: s };

    const next = { energy: s.energy - cost, lastAt: nowSec() };
    saveEnergyState(next);
    return { ok: true, state: next };
}

/**
 * Kullanıcı uygulamayı her açtığında çağırabileceğin "hydrate" yardımcıları
 */
export function getEnergy() {
    return regenEnergy(loadEnergyState()).energy;
}

export function setEnergy(v) {
    const s = loadEnergyState();
    const next = { energy: clamp(v, 0, MAX_ENERGY), lastAt: nowSec() };
    saveEnergyState(next);
    return next.energy;
}
