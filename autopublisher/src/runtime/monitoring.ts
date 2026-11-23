import { Job } from './types.js';

// The monitoring node receives operational states, never content or receipts.
export async function reportPublisherJob(env: NodeJS.ProcessEnv, job: Job): Promise<void> {
  if (job.mode !== 'publish') return;
  const origin = env.QC_PUBLISHER_MONITOR_ORIGIN?.trim();
  const token = env.QC_PUBLISHER_MONITOR_TOKEN?.trim();
  if (!origin && !token) return;
  const url = new URL(origin || '');
  if (!token || url.protocol !== 'http:' || !['127.0.0.1', '[::1]'].includes(url.hostname) ||
      url.pathname !== '/' || url.username || url.password || url.search || url.hash) {
    throw new Error('Invalid publisher monitoring configuration');
  }
  const deliveries = Object.entries(job.deliveries)
    .filter(([, delivery]) => ['confirmed', 'failed', 'uncertain'].includes(delivery.status))
    .map(([platform, delivery]) => ({platform, status: delivery.status}));
  const response = await fetch(new URL('/api/monitoring/publisher', url), {
    method: 'POST',
    redirect: 'error',
    signal: AbortSignal.timeout(2000),
    headers: {'Content-Type': 'application/json', 'X-QC-PUBLISHER-MONITOR': token},
    body: JSON.stringify({
      version: 'QC_PUBLISHER_REPORT_V1',
      jobId: job.id,
      editorial: job.editorial || 'pending',
      deliveries,
    }),
  });
  await response.body?.cancel();
  if (!response.ok) throw new Error('Publisher monitoring report rejected');
}
