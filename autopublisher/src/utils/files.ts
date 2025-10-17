import fs from 'fs-extra';
import path from 'path';

// Klasör oluştur (varsa atla)
export async function ensureDir(p: string) {
  await fs.ensureDir(p);
}

// Tüm dosyaları oku (filter opsiyonlu)
export async function listFiles(dir: string, ext?: string): Promise<string[]> {
  if (!await fs.pathExists(dir)) return [];
  const all = await fs.readdir(dir);
  return ext ? all.filter(f => f.endsWith(ext)).map(f => path.join(dir, f)) : all.map(f => path.join(dir, f));
}

// JSON okuma
export async function readJSON<T>(file: string): Promise<T | null> {
  if (!await fs.pathExists(file)) return null;
  return fs.readJSON(file);
}

// JSON yazma
export async function writeJSON(file: string, data: any) {
  await fs.outputJSON(file, data, { spaces: 2 });
}
