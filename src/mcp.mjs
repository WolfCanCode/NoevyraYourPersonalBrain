// Minimal MCP JSON-RPC 2.0 stdio transport: no runtime dependencies.
// Intentionally small tool surface; write requests create *candidate* memories.
import readline from 'node:readline';
import {openStore,projectFor,remember,recall,contextPack,logEvent} from './store.mjs';

const toolDefs = [
  {name:'brain_context',description:'Retrieve relevant verified memories for a task in the current project.',inputSchema:{type:'object',properties:{task:{type:'string'},limit:{type:'integer',minimum:1,maximum:30}},required:['task']}},
  {name:'brain_recall',description:'Search verified shared memory within the active project and personal scope.',inputSchema:{type:'object',properties:{query:{type:'string'},limit:{type:'integer',minimum:1,maximum:30}},required:['query']}},
  {name:'brain_remember',description:'Propose a memory for later user approval. Never treats agent claims as verified.',inputSchema:{type:'object',properties:{content:{type:'string'},type:{type:'string',enum:['identity','semantic','episodic','project','procedural','decision']},scope:{type:'string',enum:['project','personal']}},required:['content']}},
  {name:'brain_finish_task',description:'Record task outcome as an event. Does not automatically promote lessons to verified facts.',inputSchema:{type:'object',properties:{summary:{type:'string'},status:{type:'string',enum:['completed','blocked','failed']},evidence:{type:'string'}},required:['summary','status']}},
  {name:'brain_handoff',description:'Record structured work handoff between agents.',inputSchema:{type:'object',properties:{objective:{type:'string'},completed:{type:'array',items:{type:'string'}},remaining:{type:'array',items:{type:'string'}},evidence:{type:'string'}},required:['objective']}},
];

export async function serveMcp() {
  const db=openStore(), project=projectFor();
  const send=payload=>process.stdout.write(JSON.stringify(payload)+'\n');
  const line=readline.createInterface({input:process.stdin,crlfDelay:Infinity,terminal:false});
  try {
    for await (const msg of line) {
      let request;
      try { request=JSON.parse(msg); } catch { send({jsonrpc:'2.0',id:null,error:{code:-32700,message:'Parse error'}}); continue; }
      if (Array.isArray(request)) { send({jsonrpc:'2.0',id:null,error:{code:-32600,message:'Batch requests not supported'}}); continue; }
      if (request.id===undefined) continue; // notifications/initialized
      const ok=result=>send({jsonrpc:'2.0',id:request.id,result});
      const fail=(code,message)=>send({jsonrpc:'2.0',id:request.id,error:{code,message}});
      try {
        if (request.method==='initialize') {
          const requested=request.params?.protocolVersion;
          ok({protocolVersion:requested||'2025-06-18',capabilities:{tools:{listChanged:false}},serverInfo:{name:'personal-ai-brain',version:'0.1.0'}});
        } else if (request.method==='ping') ok({});
        else if (request.method==='tools/list') ok({tools:toolDefs});
        else if (request.method==='tools/call') {
          const {name,arguments:args={}}=request.params||{};
          if (!toolDefs.some(t=>t.name===name)) {fail(-32602,`Unknown tool: ${name}`);continue;}
          if (!args || typeof args!=='object' || Array.isArray(args)) {fail(-32602,'Invalid arguments');continue;}
          let data;
          if (name==='brain_context') {
            if (typeof args.task!=='string' || !args.task.trim()) throw new Error('task is required');
            data=contextPack(db,{task:args.task,projectId:project.id,limit:args.limit});
          } else if (name==='brain_recall') {
            if (typeof args.query!=='string' || !args.query.trim()) throw new Error('query is required');
            data=recall(db,{query:args.query,projectId:project.id,limit:args.limit});
          } else if (name==='brain_remember') {
            if(args.scope && !['personal','project'].includes(args.scope)) throw new Error('Invalid scope');
            // Disallow agent-sourced global scope writes in MVP.
            if(args.scope==='personal') throw new Error('Personal memories require CLI user approval');
            data=remember(db,{content:args.content,type:args.type||'semantic',scope:'project',projectId:project.id,source:'mcp:agent',status:'candidate'});
          } else if (name==='brain_finish_task') {
            if(typeof args.summary!=='string'||!['completed','blocked','failed'].includes(args.status)) throw new Error('Invalid task outcome');
            data={eventId:logEvent(db,'task_finish',project.id,{summary:args.summary,status:args.status,evidence:args.evidence||null})};
          } else if (name==='brain_handoff') {
            if(typeof args.objective!=='string'||!args.objective.trim()) throw new Error('objective required');
            data={eventId:logEvent(db,'handoff',project.id,{objective:args.objective,completed:args.completed||[],remaining:args.remaining||[],evidence:args.evidence||null})};
          }
          ok({content:[{type:'text',text:JSON.stringify(data,null,2)}]});
        } else fail(-32601,`Method not found: ${request.method}`);
      } catch(err) {fail(-32602,err.message);}
    }
  } finally {db.close();}
}
