import React, { useState } from "react";
import NavBar from "./components/NavBar";
import Mining from "./pages/Mining";
import Inventory from "./pages/Inventory";
import Leaderboard from "./pages/Leaderboard";
import Tasks from "./pages/Tasks";
import Links from "./pages/Links";
import Settings from "./pages/Settings";

export default function App() {
    const [tab, setTab] = useState("mining");

    let content = null;
    if (tab === "mining") content = <Mining />;
    else if (tab === "inventory") content = <Inventory />;
    else if (tab === "leaderboard") content = <Leaderboard />;
    else if (tab === "tasks") content = <Tasks />;
    else if (tab === "links") content = <Links />;
    else if (tab === "settings") content = <Settings />;

    return (
        <div style={{ width: "100vw", height: "100vh", background: "#090b14" }}>
            <NavBar active={tab} onChange={setTab} />
            {content}
        </div>
    );
}
