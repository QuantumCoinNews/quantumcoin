import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import {validateEnvironment} from '../dist/runtime/runtimeEnvironment.js';
import {configuration} from '../dist/runtime/config.js';
const fixture=JSON.parse(await fs.readFile(new URL('../../config/runtime-environment.cases.json',import.meta.url),'utf8'));
for(const c of fixture.cases)test('shared environment: '+c.name,()=>{
 const before=structuredClone(c.environment);
 assert.deepEqual(validateEnvironment(c.environment,c.service),c.errors);
 assert.deepEqual(c.environment,before);
});
test('publisher configuration rejects conflicting credential aliases without exposing values',()=>{
 const env={X_APP_KEY:'test-only-app-A',TWITTER_API_KEY:'test-only-app-B'};
 assert.throws(()=>configuration(env),e=>e.code.includes('X_APP_KEY')&&!e.message.includes('test-only-app-A')&&!e.message.includes('test-only-app-B'));
});
