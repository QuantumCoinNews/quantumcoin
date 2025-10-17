import * as THREE from "three";

/**
 * Create a large lunar ground plane.
 * - Uses textures if present under /assets/backgrounds/*
 * - Falls back to plain gray material if textures are missing.
 * - Receives shadows.
 *
 * @param {number} y Pos Y for the ground (default -2)
 * @returns {THREE.Mesh} Ground mesh
 */
export function createLunarSurface(y = -2) {
    const size = 200;       // width & height (world units)
    const segs = 256;       // segments (needed if displacementMap is present)
    const geo = new THREE.PlaneGeometry(size, size, segs, segs);

    const mat = new THREE.MeshStandardMaterial({
        color: 0x444444,
        roughness: 1.0,
        metalness: 0.0
    });

    const ground = new THREE.Mesh(geo, mat);
    ground.rotation.x = -Math.PI / 2; // make it horizontal (XZ plane)
    ground.position.y = y;
    ground.receiveShadow = true;

    // Try to load textures (optional, non-blocking)
    const loader = new THREE.TextureLoader();

    // Albedo / color
    loader.load(
        "/assets/backgrounds/moon.jpg",
        (tex) => {
            tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
            tex.repeat.set(3, 3);
            mat.map = tex;
            mat.needsUpdate = true;
        },
        undefined,
        () => {
            // ignore error: keep default gray
        }
    );

    // Normal map (optional)
    loader.load(
        "/assets/backgrounds/moon_normal.jpg",
        (tex) => {
            tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
            tex.repeat.set(3, 3);
            mat.normalMap = tex;
            mat.needsUpdate = true;
        },
        undefined,
        () => { }
    );

    // Roughness map (optional)
    loader.load(
        "/assets/backgrounds/moon_roughness.jpg",
        (tex) => {
            tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
            tex.repeat.set(3, 3);
            mat.roughnessMap = tex;
            mat.roughness = 1.0; // emphasize texture-driven roughness
            mat.needsUpdate = true;
        },
        undefined,
        () => { }
    );

    // Displacement (height) map (optional)
    loader.load(
        "/assets/backgrounds/moon_height.png",
        (tex) => {
            tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
            tex.repeat.set(3, 3);
            mat.displacementMap = tex;
            mat.displacementScale = 1.2; // crater depth
            mat.needsUpdate = true;
        },
        undefined,
        () => { }
    );

    return ground;
}
