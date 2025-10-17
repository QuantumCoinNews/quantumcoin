import React from "react";

const tabs = [
    { key: "mining", label: "Mining" },
    { key: "inventory", label: "Inventory" },
    { key: "leaderboard", label: "Leaderboard" },
    { key: "tasks", label: "Tasks" },
    { key: "links", label: "Links" },
    { key: "settings", label: "Settings" }
];

export default function NavBar({ active, onChange }) {
    return (
        <div
            style={{
                position: "absolute",
                top: 12,
                left: 12,
                right: 12,
                zIndex: 10,
                display: "flex",
                gap: 10,
                alignItems: "center",
                color: "#e8ecff"
            }}
        >
            <div style={{ fontWeight: 800, marginRight: 12 }}>🚀 Quantum Coin</div>

            {tabs.map((t) => (
                <button
                    key={t.key}
                    onClick={() => onChange(t.key)}
                    style={{
                        padding: "6px 10px",
                        borderRadius: 10,
                        border: "1px solid rgba(255,255,255,0.15)",
                        background:
                            active === t.key
                                ? "rgba(255,255,255,0.12)"
                                : "rgba(255,255,255,0.06)",
                        color: "#e8ecff",
                        cursor: "pointer"
                    }}
                >
                    {t.label}
                </button>
            ))}

            <div style={{ marginLeft: "auto", opacity: 0.75, fontSize: 12 }}>
                English UI • Testnet build
            </div>
        </div>
    );
}
