import React, { useEffect, useRef, useState } from "react";
import Scene3D from "../three/Scene3D";
import {
    BLOCK_TIME_SECONDS,
    MINING_SECONDS,
    BLOCK_REWARD_QC,
} from "../game/constants.js";
import { getStoredAddress, getWalletBalances } from "../api/wallet";
import { postProofOfPlay, epochSec } from "../api/mining";

function short(addr) {
    if (!addr) return "";
    const s = String(addr).trim();
    return s.length <= 10 ? s : `${s.slice(0, 6)}…${s.slice(-4)}`;
}

export default function Mining() {
    const wrapRef = useRef(null);
    const sceneRef = useRef(null);

    const [confirmed, setConfirmed] = useState(0);
    const [pending, setPending] = useState(0);
    const [countdown, setCountdown] = useState(BLOCK_TIME_SECONDS);
    const [blockIndex, setBlockIndex] = useState(0);
    const [status, setStatus] = useState("Ready");
    const [address, setAddress] = useState("");

    // Scene kurulum
    useEffect(() => {
        const el = wrapRef.current;
        sceneRef.current = new Scene3D(el, {
            onMiningStart: ({ seconds }) =>
                setStatus(`Mining ⛏️ (${seconds}s)`),
            onMiningComplete: ({ rewardQC }) =>
                setStatus(`Mining done (${rewardQC} QC visual)`),
        });

        // ilk hedef: rastgele saha
        // (Scene3D içindeki MoonWorld üzerinden seçer)
        setTimeout(() => {
            sceneRef.current?.setTargetSiteRandom();
            setStatus("Heading to site…");
        }, 100);

        return () => sceneRef.current?.dispose();
    }, []);

    // Adres + bakiye
    useEffect(() => {
        const addr = getStoredAddress();
        setAddress(addr || "");
        if (addr) {
            getWalletBalances(addr)
                .then((d) => {
                    setConfirmed(Number(d?.confirmed || 0));
                    setPending(Number(d?.pending || 0));
                })
                .catch(() => { });
        }
    }, []);
    useEffect(() => {
        if (!address) return;
        const id = setInterval(() => {
            getWalletBalances(address)
                .then((d) => {
                    setConfirmed(Number(d?.confirmed || 0));
                    setPending(Number(d?.pending || 0));
                })
                .catch(() => { });
        }, 15000);
        return () => clearInterval(id);
    }, [address]);

    // Blok zamanlayıcı: her blok başında yeni site seç
    useEffect(() => {
        let last = Date.now();
        const t = setInterval(() => {
            const now = Date.now();
            const dt = Math.max(0, Math.round((now - last) / 1000));
            last = now;

            setCountdown((prev) => {
                const next = prev - (dt || 1);
                if (next <= 0) {
                    onBlockComplete();
                    setBlockIndex((b) => b + 1);
                    // yeni blok başlarken yeni hedef seç
                    sceneRef.current?.setTargetSiteRandom();
                    setStatus("Heading to next site…");
                    return BLOCK_TIME_SECONDS;
                }
                return next;
            });
        }, 1000);
        return () => clearInterval(t);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [address]);

    // Blok tamamlandığında backend/mock’a gönder
    const onBlockComplete = async () => {
        const site = sceneRef.current?.currentSite;
        const blockAt = epochSec();

        // Optimistic pending
        setPending((p) => Number((p + BLOCK_REWARD_QC).toFixed(3)));

        if (!address) {
            setStatus("No address set — open Settings");
            return;
        }

        try {
            await postProofOfPlay({
                address,
                zoneId: site?.id || "site",
                blockAt,
                reward: BLOCK_REWARD_QC,
            });
            const d = await getWalletBalances(address);
            setConfirmed(Number(d?.confirmed || 0));
            setPending(Number(d?.pending || 0));
            setStatus(`Block submitted ✓ (${site?.id || "site"})`);
        } catch (e) {
            // revert optimistic if failed
            setPending((p) => Math.max(0, Number((p - BLOCK_REWARD_QC).toFixed(3))));
            setStatus(`Submit failed: ${e?.message || "error"}`);
        }
    };

    // UI
    return (
        <div style={{ width: "100%", height: "100%", background: "#090b14", overflow: "hidden" }}>
            <div
                style={{
                    position: "absolute",
                    top: 56,
                    left: 16,
                    zIndex: 9,
                    display: "flex",
                    gap: 12,
                    alignItems: "center",
                    color: "#e8ecff",
                    flexWrap: "wrap",
                    textShadow: "0 1px 2px #000",
                }}
            >
                <div>QC (Confirmed): <b>{confirmed.toFixed(3)}</b></div>
                <div>Pending: <b>{pending.toFixed(3)}</b></div>
                <div style={{ opacity: 0.9 }}>{status}</div>
                <div style={{ opacity: 0.8 }}>
                    Next block in: <b>{countdown}s</b>
                </div>
                <div style={{ opacity: 0.8 }}>
                    Address: <b>{address ? short(address) : "Not set"}</b>
                </div>
                <div style={{ opacity: 0.7 }}>
                    Reward/Block: <b>{BLOCK_REWARD_QC} QC</b> • Mining anim: <b>{MINING_SECONDS}s</b>
                </div>
            </div>

            <div ref={wrapRef} style={{ width: "100%", height: "100%" }} />
        </div>
    );
}
