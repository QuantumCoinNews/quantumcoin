import {editorialSources,projectStatus} from './sources.js';
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { Config, Platform, PublisherError, platforms } from './types.js';
import { validateEnvironment } from './runtimeEnvironment.js';

export const root = fileURLToPath(new URL('../..', import.meta.url));
export function configuration(env: NodeJS.ProcessEnv = process.env): Config {
  const absolute = (value: string | undefined, fallback: string): string => {
    if (value && !path.isAbsolute(value)) throw new PublisherError('ABSOLUTE_CONFIGURATION_PATH_REQUIRED');
    return path.resolve(value || fallback);
  };
  if(env.CONTENT_LANG && env.CONTENT_LANG!=='en')throw new PublisherError('ENGLISH_CONTENT_REQUIRED');
  const selected = (env.QC_PUBLISHER_PLATFORMS || 'youtube,x,telegram').split(',').map(x => x.trim());
  if (selected.length === 0 || new Set(selected).size !== selected.length || selected.some(x => !platforms.includes(x as Platform))) throw new PublisherError('INVALID_PLATFORMS');
  const timezone = env.QC_PUBLISHER_TIMEZONE || 'Europe/Istanbul';
  try { new Intl.DateTimeFormat('en', { timeZone: timezone }).format(); } catch { throw new PublisherError('INVALID_TIMEZONE'); }
  if(env.QC_PUBLISHER_CRON&&env.QC_PUBLISHER_CRON!=='0 18,21 * * *')throw new PublisherError('NON_CANONICAL_SCHEDULE');
  if(env.QC_PUBLISHER_SCHEDULER==='windows-task'&&timezone!=='Europe/Istanbul')throw new PublisherError('WINDOWS_SCHEDULE_TIMEZONE_MISMATCH');
  const privacy = env.YOUTUBE_PRIVACY || 'unlisted';
  if (!['private', 'unlisted', 'public'].includes(privacy)) throw new PublisherError('INVALID_PRIVACY');
  const stateDir=absolute(env.QC_PUBLISHER_STATE_DIR,path.join(env.LOCALAPPDATA||path.join(os.homedir(),'.local','share'),'QuantumCoin','autopublisher'));
  const result: Config = { root, stateDir,
    topics: absolute(env.QC_PUBLISHER_TOPICS, path.join(root, 'src/content/pool/topics_pool.json')),
    audioDir: absolute(env.QC_PUBLISHER_AUDIO_DIR, path.join(stateDir, 'media/audio')),
    backgroundDir: absolute(env.QC_PUBLISHER_BACKGROUND_DIR, path.join(stateDir, 'media/bgs')),
    timezone, cron: env.QC_PUBLISHER_CRON || '0 18,21 * * *', privacy: privacy as Config['privacy'], platforms: selected as Platform[], env };
  const issues=validateEnvironment(env,'publisher');
  if(Object.keys(issues).length)throw new PublisherError('RUNTIME_CONFIG_INVALID:'+Object.keys(issues).sort().join(','));
  return result;
}
export function localDate(now: Date, timezone: string): string {
  const parts = new Intl.DateTimeFormat('en-CA', {timeZone: timezone,year:'numeric',month:'2-digit',day:'2-digit'}).formatToParts(now);
  const part = (type: string) => parts.find(x => x.type === type)?.value;
  return `${part('year')}-${part('month')}-${part('day')}`;
}
export function validDate(value: string): boolean { return /^\d{4}-\d{2}-\d{2}$/.test(value) && !Number.isNaN(Date.parse(`${value}T00:00:00Z`)) && new Date(`${value}T00:00:00Z`).toISOString().slice(0,10) === value; }
export function configHash(c: Config): string { return createHash('sha256').update(JSON.stringify([editorialSources,projectStatus,c.topics,c.audioDir,c.backgroundDir,c.platforms,c.privacy,c.timezone,c.env.QC_PUBLISHER_AI_ENABLED||'0',c.env.QC_PUBLISHER_AI_VISUALS||'0',c.env.OPENAI_IMAGE_MODEL||'gpt-image-1',c.env.OPENAI_MODEL||'gpt-4o-mini',c.env.OPENAI_TTS_MODEL||'gpt-4o-mini-tts',c.env.OPENAI_TTS_VOICE||'alloy'])).digest('hex'); }
