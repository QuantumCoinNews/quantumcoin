import * as THREE from "three";

/**
 * Dönen Ay küresi + yüzeyde eşit dağılmış "saha" işaretleri
 * Dokular, varsa otomatik alınır:
 *  - /assets/backgrounds/moon.jpg           (albedo/rengi)
 *  - /assets/backgrounds/moon_normal.jpg    (opsiyonel)
 *  - /assets/backgrounds/moon_roughness.jpg (opsiyonel)
 *  - /assets/backgrounds/moon_height.png    (opsiyonel displacement)
 */
export function createMoonWorld({
    radius = 12,
    rotationSpeed = 0.08, // radyan/sn (yavaş döner)
    siteCount = 6,
} = {}) {
    const group = new THREE.Group();
    group.name = "moon-world";

    // --- Ay küresi ---
    const geo = new THREE.SphereGeometry(radius, 128, 96);
    const mat = new THREE.MeshStandardMaterial({
        color: 0x888888,
        metalness: 0.0,
        roughness: 1.0,
    });
    const moon = new THREE.Mesh(geo, mat);
    moon.castShadow = false;
    moon.receiveShadow = true; // gölge alabilir
    group.add(moon);

    // Dokuları yükle (varsa)
    const L = new THREE.TextureLoader();
    const safe = (path, ok) => L.load(path, ok, undefined, () => { });
    safe("/assets/backgrounds/moon.jpg", (t) => { mat.map = t; mat.needsUpdate = true; });
    safe("/assets/backgrounds/moon_normal.jpg", (t) => { mat.normalMap = t; mat.needsUpdate = true; });
    safe("/assets/backgrounds/moon_roughness.jpg", (t) => { mat.roughnessMap = t; mat.roughness = 1.0; mat.needsUpdate = true; });
    safe("/assets/backgrounds/moon_height.png", (t) => { mat.displacementMap = t; mat.displacementScale = radius * 0.03; mat.needsUpdate = true; });

    // --- Sahalar (yüzeye eşit dağılmış noktalar) ---
    const sitesGroup = new THREE.Group();
    sitesGroup.name = "sites";
    const sites = [];

    // Fibonacci küre noktaları (eşit dağılım)
    const points = fibonacciSphere(siteCount);
    const haloTex = new THREE.TextureLoader().load(
        "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAGElEQVQoU2NkYGD4z0AEYBxVSFQG0wQAAE3cAq+o9dF7AAAAAElFTkSuQmCC"
    );

    points.forEach((p, i) => {
        const normal = new THREE.Vector3(p.x, p.y, p.z).normalize();
        const worldPos = normal.clone().multiplyScalar(radius + 0.15);

        // Parıltılı işaret (sprite)
        const smat = new THREE.SpriteMaterial({
            map: haloTex,
            color: new THREE.Color().setHSL((i * 0.18) % 1, 0.8, 0.6),
            transparent: true,
            opacity: 0.8,
        });
        const sprite = new THREE.Sprite(smat);
        sprite.scale.set(1.6, 1.6, 1);
        sprite.position.copy(worldPos);

        // Kaydet
        sites.push({ id: `site-${i + 1}`, pos: worldPos, normal });
        sprite.userData = { type: "site", id: `site-${i + 1}` };

        sitesGroup.add(sprite);
    });

    group.add(sitesGroup);

    // --- API: update/döndürme & yardımcılar ---
    function update(dt) {
        group.rotation.y += rotationSpeed * dt;
        // hafif nabız
        sitesGroup.children.forEach((s, k) => {
            const pulse = 0.8 + Math.sin(performance.now() * 0.003 + k) * 0.15;
            s.material.opacity = pulse;
            const sc = 1.6 + Math.sin(performance.now() * 0.002 + k) * 0.2;
            s.scale.set(sc, sc, 1);
        });
    }

    function getSite(i) {
        return sites[(i % sites.length + sites.length) % sites.length];
    }

    function getRandomSite() {
        return sites[Math.floor(Math.random() * sites.length)];
    }

    return { group, moon, sites, sitesGroup, update, getSite, getRandomSite, radius };
}

// Yardımcı: Fibonacci sphere (yaklaşık eşit dağılım)
function fibonacciSphere(n) {
    const pts = [];
    const offset = 2 / n;
    const inc = Math.PI * (3 - Math.sqrt(5));
    for (let i = 0; i < n; i++) {
        const y = ((i * offset) - 1) + (offset / 2);
        const r = Math.sqrt(1 - y * y);
        const phi = i * inc;
        pts.push({ x: Math.cos(phi) * r, y, z: Math.sin(phi) * r });
    }
    return pts;
}
