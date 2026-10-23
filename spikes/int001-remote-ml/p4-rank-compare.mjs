// Diagnostic observations only: no quality/whole-vector release gate implied.
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';

export function compareRanks(left,right){
  assert(left.functionalPass && right.functionalPass,'functional reports must pass');
  assert(/^[a-f0-9]{64}$/.test(left.manifestSHA256),'manifest digest required');
  assert.equal(left.manifestSHA256,right.manifestSHA256,'different input manifests');
  assert.equal(left.resourceProfile,right.resourceProfile,'different resource profiles');
  const keyed=r=>new Map(r.queries.map(q=>[q.type+':'+q.q,q]));
  const a=keyed(left),b=keyed(right);
  assert.equal(a.size,left.queries.length);assert.equal(b.size,right.queries.length);
  assert.deepEqual([...a.keys()].sort(),[...b.keys()].sort());assert(a.size>0);
  const observations=[];
  for(const [key,q] of a){
    const other=b.get(key);assert(Array.isArray(q.results)&&Array.isArray(other.results));
    assert(q.results.length>0&&other.results.length>0);
    const names=rows=>rows.map(r=>{assert(/^sample-\d{3}\./.test(r.sample));return r.sample;});
    const l=names(q.results),r=names(other.results);
    assert.equal(new Set(l).size,l.length);assert.equal(new Set(r).size,r.length);
    const sameSet=JSON.stringify([...l].sort())===JSON.stringify([...r].sort());
    const row={query:key,exactRankMatch:JSON.stringify(l)===JSON.stringify(r),sameResults:sameSet};
    if(q.type==='videos'){
      assert(sameSet,'video sets differ: scores cannot be compared completely');
      const byName=new Map(other.results.map(x=>[x.sample,x]));
      row.maxAbsoluteScoreDifference=0;row.representativeFramesMatch=true;
      for(const x of q.results){
        const y=byName.get(x.sample);assert(Number.isFinite(x.score)&&Number.isFinite(y.score));
        row.maxAbsoluteScoreDifference=Math.max(row.maxAbsoluteScoreDifference,Math.abs(x.score-y.score));
        row.representativeFramesMatch&&=JSON.stringify(x.frame)===JSON.stringify(y.frame);
      }
    }
    observations.push(row);
  }
  return {scope:'public-rank-and-video-score-diagnostic-not-quality-approval',manifestSHA256:left.manifestSHA256,observations};
}
if(process.argv[1]===fileURLToPath(import.meta.url)){
  assert.equal(process.argv.length,4,'provide two report files');
  console.log(JSON.stringify(compareRanks(...process.argv.slice(2).map(p=>JSON.parse(readFileSync(p)))),null,2));
}
