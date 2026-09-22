import test from 'node:test';
import assert from 'node:assert/strict';
import {fullSmallCoverage} from './p4-small-result.mjs';
const good=()=>({catalogAssets:100,sampleHashesUnchanged:true,steps:['semantic','faces','video-semantic'].map(kind=>({kind,state:'succeeded',coverage:{complete:true,failed:0,degraded:0,eligible:90,completed:90}})),tags:{complete:true,failed:0,eligible:90,completed:90},queries:Array.from({length:8},()=>({count:10}))});
test('complete functional report passes, not quality approval',()=>assert.equal(fullSmallCoverage(good()),true));
test('settled task with 36 failed images is not success',()=>{const r=good();Object.assign(r.steps[0].coverage,{failed:36,completed:54,complete:false});assert.equal(fullSmallCoverage(r),false);});
test('reject missing phases, degraded coverage and empty queries',()=>{for(const mutate of [r=>r.steps.pop(),r=>r.steps[1].coverage.degraded=1,r=>r.queries[0].count=0,r=>r.sampleHashesUnchanged=false,r=>r.tags.failed=1]){const r=good();mutate(r);assert.equal(fullSmallCoverage(r),false);}});
