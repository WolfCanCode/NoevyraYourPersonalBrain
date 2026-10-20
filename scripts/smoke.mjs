// Run against the exact packaged executable on every native release runner.
import assert from 'node:assert/strict';
import {execFileSync, spawnSync} from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const binary=path.resolve(process.argv[2]);
const legacy=fileURLToPath(new URL('../bin/brain.mjs',import.meta.url));
const home=fs.mkdtempSync(path.join(os.tmpdir(),'brain-smoke-'));
const cwd=fs.mkdtempSync(path.join(os.tmpdir(),'brain-project-'));
const env={...process.env,BRAIN_HOME:home};
const native=(...args)=>JSON.parse(execFileSync(binary,args,{cwd,env,encoding:'utf8'}));
const js=(...args)=>JSON.parse(execFileSync(process.execPath,[legacy,...args],{cwd,env,encoding:'utf8'}));
try {
 assert.ok(native('version').version);
 native('init'); // No timer/network side effects without --schedule.
 js('remember','legacy payment webhook','--type','decision');
 assert.equal(native('recall','payment').length,1);
 native('remember','native TypeScript preference','--scope','personal');
 assert.equal(js('recall','TypeScript').length,1);
 const messages=[
  {jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'smoke',version:'1'}}},
  {jsonrpc:'2.0',method:'notifications/initialized'},
  {jsonrpc:'2.0',id:2,method:'tools/call',params:{name:'brain_remember',arguments:{content:'candidate-only nebula'}}},
 ];
 const child=spawnSync(binary,['mcp'],{cwd,env,encoding:'utf8',input:messages.map(v=>JSON.stringify(v)).join('\n')+'\n',timeout:15000});
 assert.equal(child.status,0,child.stderr);
 const responses=child.stdout.trim().split('\n').map(v=>JSON.parse(v));
 assert.equal(responses.length,2);
 assert.equal(responses[1].result.isError,false);
 const candidate=JSON.parse(responses[1].result.content[0].text);
 assert.equal(candidate.status,'candidate');
 assert.equal(js('recall','nebula').length,0);
 execFileSync(process.execPath,[legacy,'approve',candidate.id],{cwd,env,encoding:'utf8'});
 assert.equal(native('recall','nebula').length,1);
 native('sync');
 assert.ok(native('digest','--days','1').events.some(e=>e.kind==='sync'));
 assert.equal(native('mcp-config').mcpServers.brain.command,binary);
 console.log('Native binary smoke + JavaScript SQLite interoperability passed.');
} finally {fs.rmSync(home,{recursive:true,force:true});fs.rmSync(cwd,{recursive:true,force:true});}
