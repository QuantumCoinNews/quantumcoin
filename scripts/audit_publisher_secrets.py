import json,re,subprocess
from pathlib import Path
root=Path(__file__).resolve().parents[1]
patterns={
 'openai-key':re.compile(rb'sk-(?:proj-)?[A-Za-z0-9_-]{36,}'),
 'private-key':re.compile(rb'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----'),
 'oauth-token-field':re.compile(rb'"(?:refresh_token|access_token|client_secret)"\s*:\s*"(?!fixture|example|test|new-fixture)[^"\r\n]{16,}"'),
 'env-secret':re.compile(rb'(?im)^(?:OPENAI_API_KEY|TW_CONSUMER_SECRET|TW_ACCESS_SECRET|TELEGRAM_BOT_TOKEN|X_APP_SECRET)[ \t]*=[ \t]*(?!\[REDACTED_API_KEY\])[^\s#\r\n]{16,}')}
excluded={'.git','node_modules','dist','.venv','venv','__pycache__'}
def findings(data):return [kind for kind,rx in patterns.items() if rx.search(data)]
import os
work=[]
for directory,dirs,names in os.walk(root):
 dirs[:]=[x for x in dirs if x not in excluded]
 for name in names:
  p=Path(directory)/name
  try:
   if p.stat().st_size>4*1024*1024 or p.suffix.lower() in ['.db','.zip','.mp4','.mp3','.png','.jpg','.pdf','.exe','.dll']:continue
   hit=findings(p.read_bytes())
   if hit:work.append({'path':str(p.relative_to(root)),'kinds':hit})
  except OSError:continue
history=[];objects=subprocess.run(['git','rev-list','--objects','--all'],cwd=root,capture_output=True,check=True).stdout.splitlines()
paths={}
for row in objects:
 parts=row.split(b' ',1)
 if len(parts)!=2:continue
 oid,rawpath=parts;item=rawpath.decode('utf-8','replace')
 if any(x in excluded for x in item.split('/')):continue
 paths.setdefault(oid,item)
checked=subprocess.run(['git','cat-file','--batch-check'],input=b''.join(x+b'\n' for x in paths),cwd=root,capture_output=True,check=True).stdout.splitlines()
reader=subprocess.Popen(['git','cat-file','--batch'],cwd=root,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
try:
 for line in checked:
  parts=line.split()
  if len(parts)!=3 or parts[1]!=b'blob' or not parts[2].isdigit() or int(parts[2])>4*1024*1024:continue
  oid=parts[0];reader.stdin.write(oid+b'\n');reader.stdin.flush()
  header=reader.stdout.readline().split();size=int(header[2]);data=reader.stdout.read(size);reader.stdout.read(1)
  hit=findings(data)
  if hit:history.append({'blob':oid.decode(),'path':paths[oid],'kinds':hit})
finally:
 reader.stdin.close();reader.stdout.close();reader.wait()
report={'scope':'heuristic-worktree-and-all-local-git-refs','valuesIncluded':False,'rotationPerformed':False,'historyRewritten':False,'workingTree':work,'history':history,'limitations':['Pattern audit cannot prove absence of every secret.','Remote forks, caches, release archives, chat messages and provider revocation are outside this scan.']}
out=root/'autopublisher-secret-audit.local.json'
# Write outside the repository instead of adding a sensitive path inventory to source.
import os
out=Path(os.environ.get('LOCALAPPDATA',str(Path.home()))) / 'QuantumCoin/autopublisher/secret-audit.json';out.parent.mkdir(parents=True,exist_ok=True);out.write_text(json.dumps(report,indent=2),encoding='utf-8')
print(json.dumps({'workingTreeFindings':len(work),'historyFindings':len(history),'report':str(out),'valuesIncluded':False}))
