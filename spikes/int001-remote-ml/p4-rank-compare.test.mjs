import {test} from 'node:test';
import assert from 'node:assert/strict';
import {compareRanks} from './p4-rank-compare.mjs';
const fixture=()=>({functionalPass:true,manifestSHA256:'a'.repeat(64),resourceProfile:'p4-large',queries:[{q:'person',type:'videos',results:[{sample:'sample-091.mp4',score:.5,frame:{ordinal:0}}]}]});
test('same anonymous results match regardless of database IDs',()=>{
  const a=fixture(),b=fixture();a.queries[0].ids=['one'];b.queries[0].ids=['two'];
  assert.equal(compareRanks(a,b).observations[0].exactRankMatch,true);
});
test('reports score/frame differences without inventing a quality pass',()=>{
  const a=fixture(),b=fixture();b.queries[0].results[0].score=.51;b.queries[0].results[0].frame.ordinal=1;
  const result=compareRanks(a,b);assert(result.observations[0].maxAbsoluteScoreDifference>0);
  assert.equal(result.observations[0].representativeFramesMatch,false);assert(!('pass' in result));
});
test('rejects different inputs, failed runs and missing observations',()=>{
  for(const change of [b=>b.manifestSHA256='b'.repeat(64),b=>b.functionalPass=false,b=>b.queries=[]]){
    const b=fixture();change(b);assert.throws(()=>compareRanks(fixture(),b));
  }
});
