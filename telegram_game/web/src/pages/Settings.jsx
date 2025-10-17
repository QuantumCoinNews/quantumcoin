import React, { useEffect, useMemo, useState } from "react";
import { STORAGE_KEYS } from "../game/constants";
import {
    getStoredAddress,
    setStoredAddress,
    getWalletBalances,
    sendQc
} from "../api/wallet";

function useLocalAddress() {
    const [address, setAddress] = useState(() => getStoredAddress());
    const [saved, setSaved] = useState(false);

    const save = () => {
        const s = (address || "").trim();
        setStoredAddress(s);
        setSaved(true);
        setTimeout(() => setSaved(false), 1200);
    };

    return { address, setAddress, save, saved };
}

function useBalances(address) {
    const [loading, setLoading] = useState(false);
    const [balances, setBalances] = useState({ confirmed: 0, pending: 0 });
    const canFetch = useMemo(() => !!(address && address.trim().length), [address]);

    const fetchBalances = async () => {
        if (!canFetch) return;
        try {
            setLoading(true);
            const data = await getWalletBalances(address);
            setBalances({
                confirmed: Number(data?.confirmed || 0),
                pending: Number(data?.pending || 0)
            });
        } catch (e) {
            // görsel sadelik için sessiz geçiyoruz; istersen toast eklenir
            console.warn("getWalletBalances failed:", e?.message || e);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchBalances();
        const id = setInterval(fetchBalances, 15_000);
        return () => clearInterval(id);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [address]);

    return { ...balances, loading, refresh: fetchBalances };
}

export default function Settings() {
    const { address, setAddress, save, saved } = useLocalAddress();
    const { confirmed, pending, loading, refresh } = useBalances(address);

    // --- Send form state ---
    const [to, setTo] = useState("");
    const [amount, setAmount] = useState("");
    const [note, setNote] = useState("");
    const [sending, setSending] = useState(false);
    const [sendMsg, setSendMsg] = useState("");

    const copy = async (text) => {
        try {
            await navigator.clipboard.writeText(text);
            setSendMsg("Copied!");
            setTimeout(() => setSendMsg(""), 1000);
        } catch {
            setSendMsg("Copy failed");
            setTimeout(() => setSendMsg(""), 1200);
        }
    };

    const onSend = async (e) => {
        e.preventDefault();
        setSendMsg("");
        if (!address?.trim()) {
            setSendMsg("Please set your address first.");
            return;
        }
        if (!to?.trim()) {
            setSendMsg("Recipient address required.");
            return;
        }
        const amt = Number(amount);
        if (!(amt > 0)) {
            setSendMsg("Amount must be greater than 0.");
            return;
        }

        try {
            setSending(true);
            const res = await sendQc({ from: address.trim(), to: to.trim(), amount: amt, note });
            if (res?.ok) {
                setSendMsg(`Sent! ${res.txId ? "tx: " + res.txId : ""}`);
                setAmount("");
                setNote("");
                refresh(); // bakiyeleri yenile
            } else {
                setSendMsg(res?.error || "Send failed");
            }
        } catch (err) {
            setSendMsg(err?.message || "Send failed");
        } finally {
            setSending(false);
        }
    };

    return (
        <div style={{ padding: 20, color: "#e8ecff", maxWidth: 720 }}>
            <h2 style={{ marginTop: 0 }}>Settings / Wallet</h2>

            {/* Address (Receive) */}
            <section style={sectionStyle}>
                <h3 style={h3Style}>Receive (Your Address)</h3>
                <div style={row}>
                    <input
                        value={address}
                        onChange={(e) => setAddress(e.target.value)}
                        placeholder="Your wallet address"
                        style={input}
                    />
                    <button onClick={save} style={btn}>
                        Save
                    </button>
                    <button
                        onClick={() => address && copy(address)}
                        style={{ ...btn, opacity: address ? 1 : 0.5 }}
                        disabled={!address}
                    >
                        Copy
                    </button>
                </div>
                <div style={{ fontSize: 12, opacity: 0.75, marginTop: 6 }}>
                    Stored locally. Will be used for mining rewards and transfers.
                    {saved && <span style={{ marginLeft: 8, color: "#8ef9a5" }}>Saved ✓</span>}
                </div>
            </section>

            {/* Balances */}
            <section style={sectionStyle}>
                <h3 style={h3Style}>Balances</h3>
                <div style={cardRow}>
                    <div style={card}>
                        <div style={cardLabel}>Confirmed</div>
                        <div style={cardValue}>{confirmed.toFixed(3)} QC</div>
                    </div>
                    <div style={card}>
                        <div style={cardLabel}>Pending</div>
                        <div style={cardValue}>{pending.toFixed(3)} QC</div>
                    </div>
                </div>
                <button onClick={refresh} style={btn} disabled={loading}>
                    {loading ? "Refreshing..." : "Refresh"}
                </button>
            </section>

            {/* Send */}
            <section style={sectionStyle}>
                <h3 style={h3Style}>Send</h3>
                <form onSubmit={onSend}>
                    <div style={rowCol}>
                        <label style={label}>To Address</label>
                        <input
                            value={to}
                            onChange={(e) => setTo(e.target.value)}
                            placeholder="Recipient address"
                            style={input}
                        />
                    </div>
                    <div style={rowCol}>
                        <label style={label}>Amount (QC)</label>
                        <input
                            value={amount}
                            onChange={(e) => setAmount(e.target.value)}
                            placeholder="0.0"
                            style={input}
                            type="number"
                            step="0.001"
                            min="0"
                        />
                    </div>
                    <div style={rowCol}>
                        <label style={label}>Note (optional)</label>
                        <input
                            value={note}
                            onChange={(e) => setNote(e.target.value)}
                            placeholder="Memo"
                            style={input}
                        />
                    </div>
                    <div style={{ ...row, marginTop: 8 }}>
                        <button type="submit" style={btn} disabled={sending}>
                            {sending ? "Sending..." : "Send QC"}
                        </button>
                        {sendMsg && (
                            <div style={{ marginLeft: 12, fontSize: 12, opacity: 0.85 }}>
                                {sendMsg}
                            </div>
                        )}
                    </div>
                </form>
            </section>
        </div>
    );
}

const sectionStyle = {
    marginBottom: 24,
    padding: 16,
    border: "1px solid rgba(255,255,255,0.1)",
    borderRadius: 12,
    background: "rgba(255,255,255,0.04)"
};
const h3Style = { marginTop: 0, marginBottom: 12 };
const row = { display: "flex", gap: 8, alignItems: "center" };
const rowCol = { display: "flex", flexDirection: "column", gap: 6, marginBottom: 10 };
const label = { fontSize: 12, opacity: 0.8 };
const input = {
    padding: "8px 10px",
    borderRadius: 10,
    border: "1px solid rgba(255,255,255,0.2)",
    background: "rgba(0,0,0,0.2)",
    color: "#e8ecff",
    outline: "none"
};
const btn = {
    padding: "8px 12px",
    borderRadius: 10,
    border: "1px solid rgba(255,255,255,0.2)",
    background: "rgba(255,255,255,0.08)",
    color: "#e8ecff",
    cursor: "pointer"
};
const cardRow = { display: "flex", gap: 12, marginBottom: 10, flexWrap: "wrap" };
const card = {
    flex: "0 0 200px",
    padding: 12,
    borderRadius: 12,
    border: "1px solid rgba(255,255,255,0.1)",
    background: "rgba(255,255,255,0.04)"
};
const cardLabel = { fontSize: 12, opacity: 0.75 };
const cardValue = { fontSize: 18, fontWeight: 700, marginTop: 4 };
