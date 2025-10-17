import * as THREE from "three";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader.js";
import { MINING_SECONDS } from "../game/constants";
import { createMoonWorld } from "./MoonWorld";

export default class Scene3D {
    constructor(containerEl, { onMiningStart, onMiningComplete } = {}) {
        this.container = containerEl;
        this.onMiningStart = onMiningStart || (() => { });
        this.onMiningComplete = onMiningComplete || (() => { });

        this.width = this.container.clientWidth;
        this.height = this.container.clientHeight;

        this.clock = new THREE.Clock();
        this.scene = new THREE.Scene();

        // Kamera biraz uzakta, Ay'ı bütünüyle görsün
        this.camera = new THREE.PerspectiveCamera(60, this.width / this.height, 0.1, 2000);
        this.camera.position.set(0, 8, 28);

        this.renderer = new THREE.WebGLRenderer({ antialias: true });
        this.renderer.setSize(this.width, this.height);
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        this.container.appendChild(this.renderer.domElement);

        // Işıklar
        const hemi = new THREE.HemisphereLight(0xffffff, 0x23233a, 0.9);
        this.scene.add(hemi);
        const dir = new THREE.DirectionalLight(0xffffff, 1.1);
        dir.position.set(-18, 30, 18);
        dir.castShadow = true;
        dir.shadow.mapSize.set(2048, 2048);
        dir.shadow.camera.near = 0.5;
        dir.shadow.camera.far = 300;
        dir.shadow.camera.left = -60;
        dir.shadow.camera.right = 60;
        dir.shadow.camera.top = 60;
        dir.shadow.camera.bottom = -60;
        this.scene.add(dir);

        // Arkaplanı düz koyu bırakıyoruz; dönük Ay sahneyi dolduracak
        this.scene.fog = new THREE.Fog(0x0a0a12, 80, 240);

        // === Dönen Ay dünyası (saha işaretleriyle) ===
        this.moonWorld = createMoonWorld({ radius: 12, rotationSpeed: 0.08, siteCount: 6 });
        this.scene.add(this.moonWorld.group);

        // Gemi + madencilik durumu
        this.ship = null;
        this.engineSprite = null;
        this.engineLight = null;

        this.currentSite = null;     // {id,pos,normal}
        this.shipTarget = null;      // THREE.Vector3
        this.targetNormal = null;    // THREE.Vector3
        this.shipAltitude = 0.8;     // yüzeyden yükseklik
        this.moveSpeed = 6;

        this.isMining = false;
        this.miningTimer = 0;
        this.miningRing = null;
        this.miningRingAge = 0;
        this.miningLight = null;

        // Gemi yükle (GLB -> PNG -> kutu)
        this._loadShip();

        // Sadece resize dinliyoruz (click yok)
        window.addEventListener("resize", () => this._onResize());

        // Döngü
        this._anim = this._anim.bind(this);
        this.renderer.setAnimationLoop(this._anim);
    }

    // ---- Dış API ------------------------------------------------------------

    /** Rastgele bir sahayı seçer, o noktaya uçuşu başlatır. Seçilen site'i döndürür. */
    setTargetSiteRandom() {
        const site = this.moonWorld.getRandomSite();
        this.setTargetSite(site);
        return site;
    }

    /** Verilen site'e uçuşu başlatır. */
    setTargetSite(site) {
        if (!site) return;
        this.currentSite = site;
        const pos = site.pos.clone().add(site.normal.clone().multiplyScalar(this.shipAltitude));
        this.shipTarget = pos;
        this.targetNormal = site.normal.clone();
    }

    /** Madencilik efektlerini sahneden temizle */
    clearMiningFx() {
        if (this.miningRing) { this.scene.remove(this.miningRing); this.miningRing.geometry.dispose(); this.miningRing.material.dispose(); this.miningRing = null; }
        if (this.miningLight) { this.scene.remove(this.miningLight); this.miningLight = null; }
    }

    // ---- İç detaylar --------------------------------------------------------

    _loadShip() {
        const loader = new GLTFLoader();
        loader.load(
            "/assets/ship/ship.glb",
            (gltf) => {
                this.ship = gltf.scene;
                this.ship.scale.set(1.4, 1.4, 1.4);
                this.ship.position.set(0, 0.6, 0);
                this.ship.rotation.set(0, Math.PI, 0);
                this.ship.traverse((obj) => {
                    if (obj.isMesh) obj.castShadow = true;
                });
                this._ensureEngineFx();
                this.scene.add(this.ship);
            },
            undefined,
            // GLB yoksa PNG sprite, o da yoksa kutu
            () => {
                this._loadShipSprite().catch(() => this._loadShipBox());
            }
        );
    }

    async _loadShipSprite() {
        return new Promise((resolve, reject) => {
            new THREE.TextureLoader().load(
                "/assets/ship/ship.png",
                (tex) => {
                    tex.anisotropy = Math.min(8, this.renderer.capabilities.getMaxAnisotropy());
                    const mat = new THREE.SpriteMaterial({ map: tex, transparent: true });
                    const spr = new THREE.Sprite(mat);
                    spr.scale.set(2.2, 2.2, 1);
                    spr.position.set(0, 0.8, 0);
                    this.ship = spr;
                    this.scene.add(spr);
                    this._ensureEngineFx();
                    resolve(spr);
                },
                undefined,
                () => reject(new Error("ship.png not found"))
            );
        });
    }

    _loadShipBox() {
        const geo = new THREE.BoxGeometry(1.2, 0.5, 2);
        const mat = new THREE.MeshStandardMaterial({ color: 0x66ccff, metalness: 0.6, roughness: 0.3 });
        this.ship = new THREE.Mesh(geo, mat);
        this.ship.position.set(0, 0.6, 0);
        this.ship.castShadow = true;
        this.scene.add(this.ship);
        this._ensureEngineFx();
    }

    _ensureEngineFx() {
        if (!this.ship || this.engineSprite) return;
        const tex = new THREE.TextureLoader().load(
            "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAYAAACNMs+9AAAAGElEQVQoU2NkYGD4z0AEYBxVSFQG0wQAAE3cAq+o9dF7AAAAAElFTkSuQmCC"
        );
        const spr = new THREE.Sprite(new THREE.SpriteMaterial({ map: tex, color: 0x66ccff, transparent: true, opacity: 0.0 }));
        spr.scale.set(0.8, 0.8, 1);
        spr.position.set(0, -0.15, 1.1);
        this.ship.add(spr);
        this.engineSprite = spr;

        const light = new THREE.PointLight(0x66ccff, 0.0, 6);
        light.position.set(0, -0.05, 0.9);
        this.ship.add(light);
        this.engineLight = light;
    }

    _onResize() {
        this.width = this.container.clientWidth;
        this.height = this.container.clientHeight;
        this.camera.aspect = this.width / this.height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(this.width, this.height);
    }

    _moveShip(dt) {
        if (!this.ship || !this.shipTarget) {
            // Idle: motor sönsün
            if (this.engineSprite) this.engineSprite.material.opacity = Math.max(0, this.engineSprite.material.opacity - dt * 2);
            if (this.engineLight) this.engineLight.intensity = Math.max(0, this.engineLight.intensity - dt * 4);
            return;
        }

        const current = this.ship.position.clone();
        const target = this.shipTarget.clone();
        const dir = target.clone().sub(current);
        const dist = dir.length();
        if (dist < 0.12) {
            this.ship.position.copy(target);
            this.shipTarget = null;
            this._startMining(); // hedefe vardı
            return;
        }
        dir.normalize();

        // yönelim (GLB ise Y etrafında çevir)
        const desiredRotY = Math.atan2(dir.x, dir.z);
        if (this.ship.rotation) {
            const rotDiff = desiredRotY - this.ship.rotation.y;
            this.ship.rotation.y += rotDiff * Math.min(1, dt * 5);
        }

        // hareket
        const step = dir.multiplyScalar(this.moveSpeed * dt);
        this.ship.position.add(step);

        // motor efekti
        if (this.engineSprite) {
            const pulse = 0.55 + Math.sin(performance.now() * 0.02) * 0.35;
            this.engineSprite.material.opacity = Math.min(0.95, pulse);
        }
        if (this.engineLight) this.engineLight.intensity = 1.6;
    }

    _startMining() {
        if (this.isMining) return;
        this.isMining = true;
        this.miningTimer = MINING_SECONDS;
        this.onMiningStart({ seconds: MINING_SECONDS });

        // parıltılı halka: yüzey normaline dik olacak şekilde oryantasyon
        this.clearMiningFx();
        const ring = new THREE.Mesh(
            new THREE.RingGeometry(0.25, 0.45, 64),
            new THREE.MeshBasicMaterial({ color: 0x66ccff, transparent: true, opacity: 0.85, side: THREE.DoubleSide })
        );

        const basePos = this.currentSite
            ? this.currentSite.pos.clone()
            : this.ship.position.clone();
        const normal = (this.targetNormal || new THREE.Vector3(0, 1, 0)).clone();

        ring.position.copy(basePos);
        // Ring normalini (varsayılan +Z) yüzey normaline hizala
        const q = new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 0, 1), normal);
        ring.quaternion.copy(q);
        this.scene.add(ring);
        this.miningRing = ring;
        this.miningRingAge = 0;

        // hafif ışık
        const light = new THREE.PointLight(0x66ccff, 1.4, 8);
        light.position.copy(basePos.clone().add(normal.clone().multiplyScalar(0.6)));
        this.scene.add(light);
        this.miningLight = light;
    }

    _updateMining(dt) {
        if (!this.isMining) return;

        this.miningTimer -= dt;

        // halka animasyonu
        if (this.miningRing) {
            this.miningRingAge += dt;
            const s = 1 + this.miningRingAge * 1.8;
            this.miningRing.scale.set(s, s, s);
            this.miningRing.material.opacity = Math.max(0, 0.85 - this.miningRingAge * 0.6);
            if (this.miningRing.material.opacity <= 0.02) {
                this.scene.remove(this.miningRing);
                this.miningRing.geometry.dispose();
                this.miningRing.material.dispose();
                this.miningRing = null;
            }
        }

        // ışık nabzı
        if (this.miningLight) {
            this.miningLight.intensity = 1.2 + Math.sin(performance.now() * 0.015) * 0.8;
        }

        if (this.miningTimer <= 0) {
            this.isMining = false;
            this.clearMiningFx();
            this.onMiningComplete({ rewardQC: this._rollRewardQC() });
        }
    }

    _rollRewardQC() {
        // görsel/placeholder ödül
        return Number((0.5 + Math.random() * 2.5).toFixed(3));
    }

    _anim() {
        const dt = this.clock.getDelta();

        // Kamera hafif canlı kalsın
        this.camera.position.x = Math.sin(performance.now() * 0.00025) * 1.6;
        this.camera.position.y = 8 + Math.sin(performance.now() * 0.0005) * 0.3;
        this.camera.lookAt(0, 0, 0);

        // Ay dönsün
        this.moonWorld.update(dt);

        // Uçuş + madencilik
        this._moveShip(dt);
        this._updateMining(dt);

        this.renderer.render(this.scene, this.camera);
    }

    dispose() {
        this.renderer.setAnimationLoop(null);
        this.renderer.dispose();
        this.container.removeChild(this.renderer.domElement);
        window.removeEventListener("resize", this._onResize);
        this.clearMiningFx();
    }
}
