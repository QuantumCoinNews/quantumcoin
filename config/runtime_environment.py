"""Shared environment shape/alias checks. Errors contain keys and codes only.

No dotenv loading, filesystem writes, network calls or credential echoing.
Service validators still own config files, consensus and provider activation.
"""
from ipaddress import ip_address
import json
from pathlib import Path
import re
from urllib.parse import urlsplit

SCHEMA_PATH=Path(__file__).with_name('runtime-environment.schema.json')
TRUE={'1','true','yes','on'}
FALSE={'0','false','no','off'}
NETWORK={'local':'local','local-release':'local','testnet':'testnet','testnet-local':'testnet',
         'testnet-public':'testnet','mainnet':'mainnet'}
GAME_NETWORK={**NETWORK,'main':'mainnet','production':'mainnet','prod':'mainnet'}


def _port(value):
    raw=value.removeprefix(':')
    if not re.fullmatch(r'[0-9]+',raw) or not 1<=int(raw)<=65535:raise ValueError()
    return int(raw)


def _origin(value,monitor=False):
    url=urlsplit(value)
    if (not url.hostname or url.username is not None or url.password is not None or
        url.path not in ('','/') or url.query or url.fragment or url.port==0 or any(c.isspace() for c in value)):
        raise ValueError()
    try:
        address=ip_address(url.hostname)
        loopback=address.is_loopback and getattr(address,'ipv4_mapped',None) is None
    except ValueError:loopback=url.hostname=='localhost'
    if monitor:
        if url.scheme!='http' or url.hostname not in ('127.0.0.1','::1'):raise ValueError()
    elif url.scheme!='https' and not(url.scheme=='http' and loopback):raise ValueError()
    return value.rstrip('/')


def validate_environment(env,service='tooling',*,strict_unknown=True):
    schema=json.loads(SCHEMA_PATH.read_text(encoding='utf-8'))
    if schema.get('x-version')!=1 or service not in schema['x-services']:
        return {'CONFIG_SCHEMA':'unsupported-schema-or-service'}
    properties=schema['properties'];issues={}
    def get(key):
        value=env.get(key,'')
        if not isinstance(value,str):
            issues[key]='expected-string';return ''
        return value.strip()
    for key,entry in properties.items():
        if key not in env or (service!='tooling' and service not in entry['x-owners']):continue
        value=get(key)
        if not value:continue
        kind=entry['x-kind']
        try:
            if entry['x-status']=='retired':issues[key]='retired-setting';continue
            if kind=='boolean':
                if value.lower() not in TRUE|FALSE:raise ValueError()
            elif kind=='boolean-01':
                if env[key] not in {'0','1'}:raise ValueError()
            elif kind=='integer':
                if not re.fullmatch(r'-?[0-9]+',value) or not entry['minimum']<=int(value)<=entry['maximum']:raise ValueError()
            elif kind=='port':_port(value)
            elif kind=='bind':
                host,sep,port=value.rpartition(':');_port(port)
                if not sep:raise ValueError()
                if host and host.lower()!='localhost':
                    bracketed=host.startswith('[') and host.endswith(']')
                    address=ip_address(host[1:-1] if bracketed else host)
                    if bracketed!=(address.version==6):raise ValueError()
                    if getattr(address,'ipv4_mapped',None) is not None:raise ValueError()
            elif kind in ('origin','monitor-origin'):_origin(value,kind=='monitor-origin')
            elif kind=='absolute-path':
                if not(value.startswith('/') or re.match(r'^[A-Za-z]:[\\/]',value) or re.match(r'^\\\\[^\\/]+[\\/][^\\/]+',value)):raise ValueError()
            elif kind=='network':
                if value.lower() not in NETWORK:raise ValueError()
            elif kind=='game-network':
                if value.lower() not in GAME_NETWORK:raise ValueError()
            elif kind=='enum':
                candidate=value.upper() if entry.get('x-case')=='upper' else value
                if candidate not in entry['enum']:raise ValueError()
            elif kind=='platforms':
                names=[n.strip() for n in value.split(',')]
                if len(set(names))!=len(names) or any(n not in {'youtube','x','telegram','instagram','tiktok'} for n in names):raise ValueError()
            elif kind=='timezone':
                if value!='UTC' and not re.fullmatch(r'[A-Za-z_]+(?:/[A-Za-z_+0-9-]+){1,2}',value):raise ValueError()
            elif kind!='text':raise ValueError()
        except (ValueError,OverflowError):issues[key]='invalid-'+kind
    if strict_unknown:
        for key in env:
            if key.upper().startswith('QC_') and key!=key.upper():
                issues[key]='noncanonical-key-case'
            elif key.startswith('QC_') and key not in properties and not key.startswith('QC_TEST_'):
                issues[key]='unknown-setting'
    for group in schema['x-aliasGroups']:
        if service!='tooling' and service not in group['owners']:continue
        present={k:get(k) for k in group['keys'] if get(k)}
        values={v.rstrip('/') if group['normalize']=='origin' else v for v in present.values()}
        if group['normalize']=='port':
            try:values={_port(v) for v in present.values()}
            except ValueError:pass
        if group['normalize']=='game-mode':
            values={('simulated' if v.upper() in {'TESTNET','SIMULATED','TESTNET_SIM','TESTNET_SIMULATED'}
                     else 'disabled' if v.upper() in {'OFF','DISABLE','DISABLED','NONE'} else v.upper()) for v in present.values()}
        if len(values)>1:
            for key in present:issues[key]='conflicting-alias'
    if service in {'node','game','tooling'}:
        selections={}
        for key in schema['x-networkKeys']:
            value=get(key).lower()
            if value:
                mode=(GAME_NETWORK if key.startswith('TELEGRAM_GAME_') else NETWORK).get(value)
                if mode:selections[key]=mode
                else:issues[key]='invalid-network'
        for key in ('QC_TESTNET','QC_PUBLIC_TESTNET','QC_API_PUBLIC_TESTNET','QC_TESTNET_PUBLIC_API','QC_MAINNET'):
            if get(key).lower() in TRUE:selections[key]='mainnet' if key=='QC_MAINNET' else 'testnet'
        magic=get('QC_NET_MAGIC').upper()
        if magic:
            mode={'QC_LOCAL_RELEASE_V1':'local','QC_TESTNET_V1':'testnet','QC_MAINNET_V1':'mainnet'}.get(magic)
            if mode:selections['QC_NET_MAGIC']=mode
            else:issues['QC_NET_MAGIC']='invalid-enum'
        if len(set(selections.values()))>1:
            for key in selections:issues[key]='conflicting-network'
        for origin_key in ('QC_API_BASE','QUANTUMCOIN_API_BASE','QC_NODE_API'):
            value=get(origin_key)
            if value and len(set(selections.values()))==1 and origin_key not in issues:
                try:
                    port=urlsplit(value).port
                    mode=next(iter(selections.values()))
                    if (mode=='testnet' and port in (8081,8082) and get('TELEGRAM_GAME_ALLOW_TESTNET_MAIN_PORT').lower() not in TRUE) or (mode!='testnet' and port==18082):
                        issues[origin_key]='network-port-conflict'
                except ValueError:pass
    if service in {'node','tooling'}:
        bind=get('QC_API_LISTEN')
        if bind and 'QC_API_LISTEN' not in issues:
            host,_,port=bind.rpartition(':');host=host.strip('[]')
            try:local=host.lower()=='localhost' or ip_address(host).is_loopback
            except ValueError:local=False
            if not local and not get('QC_API_TOKEN'):issues['QC_API_TOKEN']='required-for-non-loopback-bind'
            if not local and get('QC_API_TOKEN').upper().startswith(('__SET_','YOUR_','CHANGE_ME')):
                issues['QC_API_TOKEN']='placeholder-credential'
            if get('QC_HTTP_PORT') and 'QC_HTTP_PORT' not in issues and _port(port)!=_port(get('QC_HTTP_PORT')):
                issues['QC_API_LISTEN']='http-port-conflict'
    if service in {'publisher','tooling'} and get('QC_PUBLISHER_MONITOR_ORIGIN') and not get('QC_PUBLISHER_MONITOR_TOKEN'):
        issues['QC_PUBLISHER_MONITOR_TOKEN']='required-with-monitor-origin'
    return dict(sorted(issues.items()))
