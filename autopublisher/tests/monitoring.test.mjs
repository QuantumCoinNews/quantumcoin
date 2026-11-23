import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {once} from 'node:events';
import {test} from 'node:test';
import {reportPublisherJob} from '../dist/runtime/monitoring.js';

test('publisher sends only operational fields to the direct node listener', async t => {
  const requests = [];
  const server = createServer(async (req, res) => {
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    requests.push({url: req.url, token: req.headers['x-qc-publisher-monitor'],
      body: JSON.parse(Buffer.concat(chunks).toString('utf8'))});
    res.writeHead(200).end();
  });
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  t.after(() => new Promise(resolve => server.close(resolve)));
  const env = {QC_PUBLISHER_MONITOR_ORIGIN: `http://127.0.0.1:${server.address().port}`,
    QC_PUBLISHER_MONITOR_TOKEN: 'test-only-publisher-token'};
  const job = {mode: 'publish', id: '2026-11-10.slot2', editorial: 'passed', pool: {title: 'internal title'},
    deliveries: {youtube: {status: 'confirmed', receipt: {id: 'internal-id', url: 'internal-url'}},
      x: {status: 'uncertain', error: 'internal-error'}, telegram: {status: 'pending'}}};
  await reportPublisherJob(env, {...job, mode: 'dry-run'});
  await reportPublisherJob(env, {...job, mode: 'render'});
  await assert.rejects(reportPublisherJob({...env, QC_PUBLISHER_MONITOR_ORIGIN: 'https://example.invalid'}, job));
  await reportPublisherJob(env, job);
  assert.deepEqual(requests, [{url: '/api/monitoring/publisher', token: env.QC_PUBLISHER_MONITOR_TOKEN, body: {
    version: 'QC_PUBLISHER_REPORT_V1', jobId: job.id, editorial: 'passed',
    deliveries: [{platform: 'youtube', status: 'confirmed'}, {platform: 'x', status: 'uncertain'}],
  }}]);
});
