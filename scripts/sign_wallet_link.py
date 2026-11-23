"""Offline wallet linkage signer. No network, keys in arguments or key storage.

Usage: python scripts/sign_wallet_link.py path/to/challenge.json
Requires cryptography. Review the printed public scope before entering your key.
"""
import getpass
import hashlib
import json
from pathlib import Path
import re
import sys
import time

sys.path.insert(0,str(Path(__file__).resolve().parents[1]/"telegram_game"))
from telegram_game.wallet_link import DOMAIN
from telegram_game.utils.nft_wallet_proof_v1 import canonical
from telegram_game.utils.nft_mint_bridge_v1 import _wallet_owner
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec


def validate_challenge(challenge, now):
    expected={"version","networkID","genesisHex","playerId","address","walletOwner","nonce","issuedAt","expiresAt"}
    if not isinstance(challenge,dict) or set(challenge)!=expected or challenge["version"]!="QC_GAME_WALLET_LINK_V1":
        raise ValueError("Invalid challenge schema")
    if not re.fullmatch(r"tg_[0-9]{1,20}",challenge["playerId"]) or not re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}",challenge["networkID"]) or not re.fullmatch(r"[0-9a-f]{64}",challenge["genesisHex"]) or not re.fullmatch(r"[0-9a-f]{64}",challenge["nonce"]):
        raise ValueError("Invalid challenge identity")
    if type(challenge["issuedAt"]) is not int or type(challenge["expiresAt"]) is not int or not challenge["issuedAt"]<=now<challenge["expiresAt"] or challenge["expiresAt"]-challenge["issuedAt"]!=300:
        raise ValueError("Expired challenge")
    if challenge["walletOwner"] != _wallet_owner(challenge["address"]):
        raise ValueError("Invalid challenge address")


def sign(challenge, secret, now):
    validate_challenge(challenge, now)
    if not re.fullmatch(r"[0-9a-fA-F]{64}",secret):
        raise ValueError("Invalid private key")
    key=ec.derive_private_key(int(secret,16),ec.SECP256K1())
    public=key.public_key().public_bytes(serialization.Encoding.X962,serialization.PublicFormat.UncompressedPoint)
    owner=hashlib.new("ripemd160",hashlib.sha256(public[1:]).digest()).hexdigest()
    if owner!=challenge["walletOwner"] or owner!=_wallet_owner(challenge["address"]):
        raise ValueError("This key does not control the challenge address")
    signature=key.sign(DOMAIN+canonical(challenge).encode("ascii"),ec.ECDSA(hashes.SHA256()))
    return {"playerId":challenge["playerId"],"address":challenge["address"],"nonce":challenge["nonce"],"publicKey":public.hex(),"signature":signature.hex()}


if __name__=="__main__":
    try:
        if not sys.stdin.isatty():
            raise ValueError("An interactive terminal is required for hidden key input")
        source=Path(sys.argv[1])
        if source.stat().st_size>8192:raise ValueError("Oversized challenge")
        challenge=json.loads(source.read_text(encoding="utf-8"))
        validate_challenge(challenge,int(time.time()))
        print("Review the wallet address, Telegram account, network and genesis:",file=sys.stderr)
        print(json.dumps(challenge,indent=2),file=sys.stderr)
        print("Type SIGN to approve this wallet linkage:",file=sys.stderr)
        if input()!="SIGN":raise ValueError("Cancelled")
        secret=getpass.getpass("Wallet private key (local input only): ")
        proof=sign(challenge,secret,int(time.time()))
        secret=""
        print(json.dumps(proof,separators=(",",":")))
    except Exception:
        print("Signing failed; check challenge scope, expiry and wallet key. No key was sent or saved.",file=sys.stderr)
        raise SystemExit(1)
