"""Run isolated NFT regression scripts and preserve exact commands/results.

No historical installers, source writes, real application DBs or live nodes.
Use a project venv with requirements-nft-evidence.txt already installed.
"""
import argparse
import concurrent.futures
import datetime
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import time


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("--output",type=Path,required=True)
    parser.add_argument("--contract",type=Path,required=True)
    args=parser.parse_args()
    root=Path(__file__).resolve().parents[1]
    output=args.output.resolve()
    output.mkdir(parents=True,exist_ok=True)
    contract=args.contract.resolve()
    if not contract.is_file():
        parser.error("Generate the Go TestNFTVerifiedLifecycleContractV1 contract first")
    files=sorted((root/"telegram_game/tests").glob("test_*.py"))
    files.append(root/"scripts/test_monthly_nft_catalog_v1.py")
    manifest={}
    for directory in ("telegram_game/telegram_game","telegram_game/tests","nft","node","scripts"):
        for path in (root/directory).rglob("*"):
            if path.is_file() and path.suffix in {".py",".go",".json",".js"} and "__pycache__" not in path.parts:
                manifest[path.relative_to(root).as_posix()]=hashlib.sha256(path.read_bytes()).hexdigest()
    (output/"source-manifest.json").write_text(json.dumps(manifest,indent=2,sort_keys=True),encoding="utf-8")
    env=os.environ.copy()
    env.update(QC_API_BASE="http://127.0.0.1:1",QC_API_TOKEN="test-only-unreachable-node",
               TELEGRAM_GAME_ADMIN_TOKEN="test-only-regression-admin",TELEGRAM_BOT_TOKEN="test-only-regression-bot",
               QC_ENABLE_CHAIN_NFT_MINT="0",QC_ENABLE_TG_NFT_MINT_BRIDGE="0",
               QMA_NFT_LIFECYCLE_CONTRACT_PATH=str(contract),QMA_NFT_EVIDENCE_OUTPUT_DIR=str(output/"test-only-evidence"))
    def run(path):
        command=[sys.executable,str(path)]
        start=datetime.datetime.now(datetime.timezone.utc).isoformat()
        begin=time.monotonic()
        log=output/(path.stem+".log")
        with tempfile.TemporaryDirectory(prefix="qma-regression-cwd-") as temp:
            try:
                result=subprocess.run(command,cwd=temp,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=240)
                text=result.stdout.decode("utf-8",errors="replace")
                status="PASS" if result.returncode==0 else "FAIL"
                if result.returncode==0 and re.search(r"OK \(skipped=|^.*\.\.\. skipped ",text,re.MULTILINE): status="SKIP"
                exit_code=result.returncode
            except subprocess.TimeoutExpired as error:
                text=(error.stdout or b"").decode("utf-8",errors="replace")
                status,exit_code="TIMEOUT",None
        log.write_text(text,encoding="utf-8")
        counts=re.findall(r"Ran (\d+) tests? in",text)
        return {"script":path.relative_to(root).as_posix(),"command":command,
                "workingDirectory":"isolated temporary directory","startedAtUTC":start,
                "elapsedSeconds":round(time.monotonic()-begin,3),"exitCode":exit_code,
                "result":status,"unittestCount":int(counts[-1]) if counts else None,"log":str(log)}
    results=[]
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
        for result in executor.map(run,files):
            results.append(result)
            print(result["result"],result["script"],flush=True)
            (output/"results.json").write_text(json.dumps(results,indent=2),encoding="utf-8")
    versions={name:importlib.metadata.version(name) for name in ("cryptography","py-ecc","fastapi","httpx")}
    changed=[name for name,sha in manifest.items() if not (root/name).is_file() or hashlib.sha256((root/name).read_bytes()).hexdigest()!=sha]
    summary={"python":sys.executable,"pythonVersion":sys.version,"dependencies":versions,
             "sourceChangedDuringRun":changed,"scriptCount":len(results),
             "results":{state:sum(r["result"]==state for r in results) for state in ("PASS","FAIL","SKIP","TIMEOUT")}}
    (output/"summary.json").write_text(json.dumps(summary,indent=2),encoding="utf-8")
    print(json.dumps(summary,indent=2))
    return 0 if all(r["result"]=="PASS" for r in results) and not changed else 1


if __name__=="__main__":
    raise SystemExit(main())
