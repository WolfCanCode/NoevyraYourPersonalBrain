import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { brainHome } from './store.mjs';

const platform=process.platform;
function exec(command,args){return spawnSync(command,args,{encoding:'utf8',timeout:5000});}
export function schedulerStatus(){
  const home=brainHome();
  if(platform==='darwin') return {platform,definition:path.join(os.homedir(),'Library/LaunchAgents/dev.personal-ai-brain.daily.plist'),configured:fs.existsSync(path.join(os.homedir(),'Library/LaunchAgents/dev.personal-ai-brain.daily.plist'))};
  if(platform==='linux') return {platform,definition:path.join(os.homedir(),'.config/systemd/user/personal-ai-brain.timer'),configured:fs.existsSync(path.join(os.homedir(),'.config/systemd/user/personal-ai-brain.timer'))};
  return {platform,configured:false,home,message:'Use Task Scheduler on Windows (not installed by v0.1.0)'};
}
export function installScheduler(executable){
  const node=process.execPath;
  if(platform==='darwin'){
    const dir=path.join(os.homedir(),'Library/LaunchAgents');fs.mkdirSync(dir,{recursive:true});
    const file=path.join(dir,'dev.personal-ai-brain.daily.plist');
    const xml=s=>s.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;');
    const plist=`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd"><plist version="1.0"><dict><key>Label</key><string>dev.personal-ai-brain.daily</string><key>ProgramArguments</key><array><string>${xml(node)}</string><string>${xml(executable)}</string><string>sync</string><string>--online</string></array><key>StartCalendarInterval</key><dict><key>Hour</key><integer>8</integer><key>Minute</key><integer>0</integer></dict><key>StandardOutPath</key><string>${xml(path.join(brainHome(),'sync.log'))}</string><key>StandardErrorPath</key><string>${xml(path.join(brainHome(),'sync.error.log'))}</string></dict></plist>`;
    fs.writeFileSync(file,plist,{mode:0o600});
    const r=exec('launchctl',['bootstrap',`gui/${process.getuid()}`,file]);
    return {installed:r.status===0,message:r.status===0?'✓ Daily launchd timer installed (08:00 local)':'⚠ launchd config written; use launchctl bootstrap manually if needed'};
  }
  if(platform==='linux'){
    const dir=path.join(os.homedir(),'.config/systemd/user');fs.mkdirSync(dir,{recursive:true});
    const service=`[Unit]\nDescription=Noevyra daily maintenance\n[Service]\nType=oneshot\nExecStart=${node} ${executable} sync --online\n`;
    const timer='[Unit]\nDescription=Daily Noevyra maintenance\n[Timer]\nOnCalendar=*-*-* 08:00:00\nPersistent=true\n[Install]\nWantedBy=timers.target\n';
    fs.writeFileSync(path.join(dir,'personal-ai-brain.service'),service);
    fs.writeFileSync(path.join(dir,'personal-ai-brain.timer'),timer);
    exec('systemctl',['--user','daemon-reload']);
    const r=exec('systemctl',['--user','enable','--now','personal-ai-brain.timer']);
    return {installed:r.status===0,message:r.status===0?'✓ Daily systemd timer installed (08:00 local)':'⚠ User systemd unavailable; timer files generated. Run brain sync manually until enabled.'};
  }
  return {installed:false,message:'⚠ Automatic schedule setup is available for macOS/Linux; configure Windows Task Scheduler manually.'};
}
