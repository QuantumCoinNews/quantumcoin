import { api } from "./client";
import { STORAGE_KEYS, BATCH_WRITE_SECONDS } from "../game/constants";
import { epochSec } from "./mining";

// --- Local address management ---

export function getStoredAddress() {
    try {
        const raw = window.localStorage.getItem(STORAGE_KEYS.ADDRESS);
        return raw ? JSON.parse(raw) : "";
    } catch {
        return "";
    }
}

export function setStoredAddress(addr) {
    const s = String(addr || "").trim();
    window.localStorage.setItem(STORAGE_KEYS.ADDRESS, JSON.stringify(s));
    return s;
}

export function hasAddress() {
    return !!getStoredAddress();
}

// --- Server endpoints (placeholders) ---
const ENDPOINTS = {
    balances: "/api/wallet/balances", // GET ?address=
    send: "/api/wallet/send"          // POST {from,to,amount,note}
};

const USE_MOCK = String(import.meta?.env?.VITE_USE_MOCK || "") === "1";

/* ------------------------------ MOCK HELPERS ------------------------------ */

function mockKey(addr) {
    const a = String(addr || "").trim();
    return `qc_mock_wallet_${a}`;
}

function mockRead(addr) {
    try {
        const raw = localStorage.getItem(mockKey(addr));
        if (raw) return JSON.parse(raw);
    } catch { }
    return { confirmed: 0, pending: 0, lastBatchAt: epochSec() };
}

function mockWrite(addr, state) {
    localStorage.setItem(mockKey(addr), JSON.stringify(state));
    return state;
}

function mockMaybeSettle(addr) {
    const now = epochSec();
    const s = mockRead(addr);
    if (!s.lastBatchAt) s.lastBatchAt = now;
    if (now - s.lastBatchAt >= BATCH_WRITE_SECONDS) {
        s.confirmed = Number((s.confirmed + s.pending).toFixed(3));
        s.pending = 0;
        s.lastBatchAt = now;
        mockWrite(addr, s);
    }
    return s;
}

/* ---------------------------------- API ---------------------------------- */

export async function getWalletBalances(address) {
    const saddr = String(address || "").trim();
    if (!saddr) throw new Error("Address required");

    if (USE_MOCK) {
        const s = mockMaybeSettle(saddr);
        return { confirmed: s.confirmed, pending: s.pending, lastBatchAt: s.lastBatchAt };
    }

    const q = new URLSearchParams({ address: saddr }).toString();
    return api.get(`${ENDPOINTS.balances}?${q}`);
}

/**
 * QC gönderimi (Gönder)
 * Beklenen yanıt: { ok: boolean, txId?: string, error?: string }
 * MOCK: sender.confirmed >= amount ise düşer, receiver.confirmed'e ekler.
 */
export async function sendQc({ from, to, amount, note }) {
    const body = {
        from: String(from || "").trim(),
        to: String(to || "").trim(),
        amount: Number(amount || 0),
        note: note ? String(note).trim() : undefined
    };
    if (!body.from) throw new Error("Sender address required");
    if (!body.to) throw new Error("Recipient address required");
    if (!(body.amount > 0)) throw new Error("Amount must be > 0");

    if (USE_MOCK) {
        const sFrom = mockMaybeSettle(body.from);
        if (sFrom.confirmed < body.amount) {
            return { ok: false, error: "Insufficient confirmed balance" };
        }
        const sTo = mockMaybeSettle(body.to);
        sFrom.confirmed = Number((sFrom.confirmed - body.amount).toFixed(3));
        sTo.confirmed = Number((sTo.confirmed + body.amount).toFixed(3));
        mockWrite(body.from, sFrom);
        mockWrite(body.to, sTo);
        const txId = `mock_${Date.now().toString(36)}`;
        return { ok: true, txId };
    }

    return api.post(ENDPOINTS.send, body);
}
