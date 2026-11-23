import fs from 'node:fs/promises';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import { PoolAsset, PublisherError } from './types.js';

export async function render(pool: PoolAsset, directory: string, assertLease: ()=>void): Promise<string> {
  await fs.mkdir(directory,{recursive:true,mode:0o700});
  const temporary=path.join(directory,`.render-${randomUUID()}.mp4`);const output=path.join(directory,'video.mp4');
  const image=/\.(png|jpe?g)$/i.test(pool.background);
  let duration:number;
  try{const result=await promisify(execFile)('ffprobe',['-v','error','-show_entries','format=duration','-of','json',pool.audio],{windowsHide:true,timeout:30000,maxBuffer:65536});duration=Number(JSON.parse(result.stdout).format?.duration);}catch{throw new PublisherError('AUDIO_DURATION_UNAVAILABLE');}
  if(!Number.isFinite(duration)||duration<=0||duration>120)throw new PublisherError('AUDIO_DURATION_OUT_OF_BOUNDS');
  const args=['-nostdin','-hide_banner','-loglevel','error','-filter_threads','1',...(image?['-loop','1','-framerate','1']:['-stream_loop','-1']),'-i',pool.background,'-i',pool.audio,
    '-map','0:v:0','-map','1:a:0','-vf','scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920','-r','30','-c:v','libx264','-threads','2','-preset','veryfast','-pix_fmt','yuv420p','-c:a','aac','-t',String(duration),'-shortest','-movflags','+faststart',temporary];
  assertLease();
  await new Promise<void>((resolve,reject)=>{
    const child=spawn('ffmpeg',args,{stdio:'ignore',windowsHide:true});
    const monitor=setInterval(()=>{try{assertLease();}catch{child.kill();}},250);
    const timer=setTimeout(()=>child.kill(),20*60*1000);
    const cleanup=()=>{clearTimeout(timer);clearInterval(monitor);};
    child.once('error',()=>{cleanup();reject(new PublisherError('RENDER_TOOL_UNAVAILABLE'));});
    child.once('exit',code=>{cleanup();code===0?resolve():reject(new PublisherError('RENDER_FAILED'));});
  });
  assertLease();
  // Only the one locked runtime writes this job directory. Interrupted .render
  // files are preserved; a new render never consumes them as valid output.
  await fs.rename(temporary,output);return output;
}
