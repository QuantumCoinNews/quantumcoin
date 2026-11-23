import fs from 'node:fs/promises';
import path from 'node:path';
import { randomUUID } from 'node:crypto';
import { PublisherError } from './types.js';

export async function readJSON<T>(file: string): Promise<T | undefined> {
  try { const stat = await fs.lstat(file); if (!stat.isFile() || stat.size > 8*1024*1024) throw new PublisherError('STATE_FILE_INVALID'); return JSON.parse(await fs.readFile(file,'utf8')) as T; }
  catch(error) { if ((error as NodeJS.ErrnoException).code === 'ENOENT') return undefined; throw new PublisherError('STATE_CORRUPT_PRESERVED'); }
}
export async function saveJSON(file: string, value: unknown): Promise<void> {
  await fs.mkdir(path.dirname(file),{recursive:true,mode:0o700});
  const temporary = path.join(path.dirname(file),`.commit-${randomUUID()}`);
  const handle = await fs.open(temporary,'wx',0o600);
  try { await handle.writeFile(JSON.stringify(value,null,2)+'\n'); await handle.sync(); } finally { await handle.close(); }
  try { await fs.rename(temporary,file); } catch { await fs.unlink(temporary).catch(()=>{}); throw new PublisherError('STATE_COMMIT_FAILED'); }
}
