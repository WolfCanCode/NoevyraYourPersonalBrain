import test,{after} from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
const home=fs.mkdtempSync(path.join(os.tmpdir(),'brain-tests-'));
process.env.BRAIN_HOME=home;
import {openStore,remember,recall,approve,contextPack,stats,logEvent,digest} from '../src/store.mjs';
after(()=>fs.rmSync(home,{recursive:true,force:true}));

test('verified memories are isolated by project',()=>{
  const db=openStore();
  try {
    remember(db,{content:'Payment webhook uses HMAC and idempotency keys',scope:'project',projectId:'vezit'});
    remember(db,{content:'Payment token should be rotated after 24 hours',scope:'project',projectId:'another'});
    const result=recall(db,{query:'Payment',projectId:'vezit'});
    assert.equal(result.length,1);
    assert.match(result[0].content,/HMAC/);
    assert.equal(recall(db,{query:'Payment',projectId:'unrelated'}).length,0);
  } finally {db.close();}
});

test('agent memories require explicit promotion',()=>{
  const db=openStore();
  try {
    const item=remember(db,{content:'We decided to use PostgreSQL for this repo',scope:'project',projectId:'vezit',status:'candidate',source:'mcp:agent'});
    assert.equal(recall(db,{query:'PostgreSQL',projectId:'vezit'}).length,0);
    assert.ok(approve(db,item.id));
    assert.equal(recall(db,{query:'PostgreSQL',projectId:'vezit'}).length,1);
    assert.equal(approve(db,item.id),false);
  } finally {db.close();}
});

test('personal memory shares across projects and context includes it',()=>{
  const db=openStore();
  try {
    remember(db,{content:'I prefer concise TypeScript solutions',scope:'personal'});
    const pack=contextPack(db,{task:'TypeScript',projectId:'newproject'});
    assert.equal(pack.memories.length,1);
    assert.equal(pack.memories[0].scope,'personal');
  } finally {db.close();}
});

test('events and stats are recorded',()=>{
  const db=openStore();try {
    logEvent(db,'handoff','vezit',{objective:'finish API'});
    assert.ok(stats(db).total>=3);
    assert.ok(digest(db,1).events.some(e=>e.kind==='handoff'));
  } finally {db.close();}
});
