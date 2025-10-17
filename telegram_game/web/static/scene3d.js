// web/static/scene3d.js
(() => {
    // --- Temel kurulum ---
    const root = document.getElementById("three-root");
    if (!root || !window.THREE) return;

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.outputEncoding = THREE.sRGBEncoding;
    renderer.setPixelRatio(Math.min(2, window.devicePixelRatio || 1));
    root.appendChild(renderer.domElement);

    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(60, 1, 0.1, 2000);
    camera.position.set(0, 28, 64);

    const controls = new THREE.OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.target.set(0, 5, 0);

    // Işıklar
    scene.add(new THREE.HemisphereLight(0x88aaff, 0x101020, 0.6));
    const dir = new THREE.DirectionalLight(0xffffff, 1.0);
    dir.position.set(30, 60, 20);
    dir.castShadow = false;
    scene.add(dir);

    // Boyutlandırma
    function resize() {
        const w = root.clientWidth, h = root.clientHeight;
        renderer.setSize(w, h, false);
        camera.aspect = w / h; camera.updateProjectionMatrix();
    }
    window.addEventListener("resize", resize, { passive: true });
    resize();

    // --- Zemin (Ay yüzeyi) ---
    const texLoader = new THREE.TextureLoader();
    const moonTex = texLoader.load("/static/img/moon_bg.jpg");
    moonTex.wrapS = moonTex.wrapT = THREE.RepeatWrapping;
    moonTex.repeat.set(1, 1);

    const ground = new THREE.Mesh(
        new THREE.PlaneGeometry(160, 90, 1, 1),
        new THREE.MeshStandardMaterial({
            map: moonTex,
            roughness: 0.95,
            metalness: 0.0
        })
    );
    ground.rotation.x = -Math.PI / 2;
    ground.position.y = 0;
    scene.add(ground);

    // --- 5 kazım noktası (markırlar) ---
    const sectorDefs = [
        { name: "Sector 1", x: -55, z: -10 },
        { name: "Sector 2", x: -27, z: -5 },
        { name: "Sector 3", x: 0, z: 0 },
        { name: "Sector 4", x: 27, z: -5 },
        { name: "Sector 5", x: 55, z: -10 },
    ];

    const markers = [];
    const markerGeo = new THREE.CylinderGeometry(1.8, 1.8, 1.0, 20);
    sectorDefs.forEach((s) => {
        const m = new THREE.Mesh(
            markerGeo,
            new THREE.MeshStandardMaterial({ color: 0x2a3158, emissive: 0x000000 })
        );
        m.position.set(s.x, 0.5, s.z);
        m.userData.sector = s.name;
        scene.add(m);
        markers.push(m);
    });

    function highlight(name) {
        markers.forEach(m => {
            const on = m.userData.sector === name;
            m.material.color.set(on ? 0xffd54a : 0x2a3158);
            m.material.emissive.set(on ? 0x664400 : 0x000000);
        });
    }

    // --- Gemi ---
    const gltfLoader = new THREE.GLTFLoader();
    let ship = null;
    function loadShipModel(src) {
        gltfLoader.load(src, (g) => {
            if (ship) scene.remove(ship);
            ship = g.scene || g.scenes?.[0];
            ship.traverse(o => {
                if (o.isMesh) {
                    o.castShadow = false; o.receiveShadow = false;
                    if (o.material) {
                        o.material.metalness = 0.1;
                        o.material.roughness = 0.6;
                    }
                }
            });
            ship.scale.set(2.2, 2.2, 2.2);
            ship.position.set(0, 4, 20);
            scene.add(ship);
        }, undefined, (err) => {
            console.warn("GLB load failed, using fallback cone.", err);
            // Fallback
            const cone = new THREE.Mesh(
                new THREE.ConeGeometry(2, 6, 16),
                new THREE.MeshStandardMaterial({ color: 0x7aa2ff, emissive: 0x0 })
            );
            cone.rotation.x = Math.PI / 2;
            ship = new THREE.Group();
            ship.add(cone);
            ship.position.set(0, 4, 20);
            scene.add(ship);
        });
    }
    loadShipModel("/static/models/ship_std.glb");

    // Gemi hedefe akıcı hareket etsin
    let targetPos = new THREE.Vector3(0, 4, 20);
    function moveShipToSector(name) {
        const s = sectorDefs.find(x => x.name === name);
        if (!s || !ship) return;
        targetPos.set(s.x, 4, s.z + 8);
    }

    // --- Raycast ile seçim ---
    const raycaster = new THREE.Raycaster();
    const mouse = new THREE.Vector2();
    renderer.domElement.addEventListener("click", (ev) => {
        const rect = renderer.domElement.getBoundingClientRect();
        mouse.x = ((ev.clientX - rect.left) / rect.width) * 2 - 1;
        mouse.y = -((ev.clientY - rect.top) / rect.height) * 2 + 1;
        raycaster.setFromCamera(mouse, camera);
        const hit = raycaster.intersectObjects(markers, false)[0];
        if (hit?.object?.userData?.sector) {
            const name = hit.object.userData.sector;
            // HUD güncelle (mevcut UI)
            const el = document.getElementById("sector");
            if (el) el.innerText = name;
            moveShipToSector(name);
            highlight(name);
            // 2D tarafın seçili sektör değişkeni varsa tutarlı kalsın:
            window.S && (window.S.selectedSector = name);
        }
    });

    // --- Parçacıklar (altın) ---
    const particleGeo = new THREE.SphereGeometry(0.15, 6, 6);
    const particleMat = new THREE.MeshStandardMaterial({ color: 0xffd54a, emissive: 0x332200 });
    const particles = [];
    function spawnBurst(x, z) {
        for (let i = 0; i < 24; i++) {
            const p = new THREE.Mesh(particleGeo, particleMat.clone());
            p.position.set(x + (Math.random() - 0.5) * 1.5, 1.2 + Math.random() * 0.6, z + (Math.random() - 0.5) * 1.5);
            p.userData.vx = (Math.random() - 0.5) * 0.12;
            p.userData.vy = Math.random() * 0.10 + 0.02;
            p.userData.vz = (Math.random() - 0.5) * 0.12;
            p.userData.life = 0.9 + Math.random() * 0.7;
            scene.add(p);
            particles.push(p);
        }
    }

    // --- App.js ile entegrasyon köprüleri ---
    window.Game3D = {
        onMiningStart: (sectorName, durationSec = 30) => {
            const s = sectorDefs.find(x => x.name === sectorName) || sectorDefs[2];
            // madencilik süresince ara ara partikül patlat
            const t0 = performance.now(), T = durationSec * 1000;
            const timer = setInterval(() => {
                const t = performance.now() - t0;
                if (t > T) { clearInterval(timer); return; }
                spawnBurst(s.x, s.z);
            }, 700);
        },
        onMiningFinish: (sectorName, rewardQC) => {
            const s = sectorDefs.find(x => x.name === sectorName) || sectorDefs[2];
            // final patlaması
            for (let k = 0; k < 3; k++) setTimeout(() => spawnBurst(s.x, s.z), k * 150);
        },
        onShipChange: (imgOrModelPath) => { // mağazadan yeni gemi alınırsa
            if (!imgOrModelPath) return;
            // Eğer model yolu verdiysek .glb bekleriz:
            if (/\.(glb|gltf)$/i.test(imgOrModelPath)) loadShipModel(imgOrModelPath);
        }
    };

    // --- Döngü ---
    const tmp = new THREE.Vector3();
    function animate() {
        requestAnimationFrame(animate);

        // ship easing
        if (ship) {
            ship.position.lerp(targetPos, 0.06);
            // hedefe bak
            tmp.copy(targetPos); tmp.y = ship.position.y;
            ship.lookAt(tmp);
        }

        // parçacık simülasyonu
        for (let i = particles.length - 1; i >= 0; i--) {
            const p = particles[i];
            p.position.x += p.userData.vx * 60 * 0.016;
            p.position.y += p.userData.vy * 60 * 0.016;
            p.position.z += p.userData.vz * 60 * 0.016;
            p.userData.vy -= 0.012;
            p.userData.life -= 0.016;
            p.material.emissiveIntensity = Math.max(0, p.userData.life);
            if (p.userData.life <= 0) {
                scene.remove(p); particles.splice(i, 1);
            }
        }

        controls.update();
        renderer.render(scene, camera);
    }
    animate();
})();
