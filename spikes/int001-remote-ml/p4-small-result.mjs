// Diagnostic acceptance only; not a production job state machine.
export function fullSmallCoverage(report) {
  const complete=c=>c?.complete===true&&c.failed===0&&(c.degraded??0)===0&&c.completed===c.eligible;
  return report.catalogAssets===100&&report.sampleHashesUnchanged===true&&
    report.steps?.length===3&&new Set(report.steps.map(s=>s.kind)).size===3&&
    ['semantic','faces','video-semantic'].every(kind=>report.steps.some(s=>s.kind===kind&&s.state==='succeeded'&&complete(s.coverage)))&&
    complete(report.tags)&&report.queries?.length===8&&report.queries.every(q=>q.count>0);
}
