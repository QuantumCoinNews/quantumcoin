export const platforms = ['youtube', 'x', 'telegram', 'instagram', 'tiktok'] as const;
export type Platform = typeof platforms[number];
export type Mode = 'dry-run' | 'render' | 'publish';
export class PublisherError extends Error {
  constructor(public readonly code: string) { super(code); this.name = 'PublisherError'; }
}
export interface Topic { id: string; title: string; script: string; hashtags?: string[]; telegram?: string; x?: string; youtube_description?: string }
export interface PoolAsset { idx: number; next: number; topic: Topic; audio: string; background: string; fingerprint: string }
export interface Receipt { id: string; url?: string }
export interface PreparedContent { topic:Topic; audio:string; audioHash:string; background?:string; backgroundHash?:string; model:string; speechModel:string; language:'en' }
export type DeliveryState = 'pending' | 'submitting' | 'confirmed' | 'failed' | 'uncertain';
export interface Delivery { status: DeliveryState; attempts: number; receipt?: Receipt; error?: string; httpStatus?: number }
export interface Job { version: 'QC_PUBLISH_JOB_V1'; id: string; date: string; slot: 1 | 2; mode: Mode; status: 'planned' | 'preparing' | 'rendering' | 'rendered' | 'publishing' | 'completed' | 'failed' | 'needs-review' | 'abandoned'; pool: PoolAsset; configHash: string; editorial?: 'passed' | 'rejected'; prepared?:PreparedContent; generationAttempts?:number; video?: string; videoHash?: string; deliveries: Partial<Record<Platform, Delivery>>; error?: string }
export interface Config { root: string; stateDir: string; topics: string; audioDir: string; backgroundDir: string; timezone: string; cron: string; privacy: 'private' | 'unlisted' | 'public'; platforms: Platform[]; env: NodeJS.ProcessEnv }
export interface PublishInput { title: string; description: string; text: string; video: string; privacy: Config['privacy'] }
export interface Provider { readiness(env: NodeJS.ProcessEnv): string | undefined; publish(input: PublishInput, env: NodeJS.ProcessEnv): Promise<Receipt> }
