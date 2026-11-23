"""Check coverage and export declarations/source references, never environment values."""
import argparse,json
from runtime_environment_inventory import ROOT,inventory

OUTPUT=ROOT/'docs/runtime-environment-catalog.json'

def catalog():
    schema=json.loads((ROOT/'config/runtime-environment.schema.json').read_text(encoding='utf-8'))
    refs=inventory();properties=schema['properties']
    missing=set(refs)-set(properties)
    if missing:raise ValueError('undeclared environment keys: '+','.join(sorted(missing)))
    kinds={'text','boolean','boolean-01','integer','port','bind','origin','monitor-origin','absolute-path','network','game-network','enum','platforms','timezone'}
    rows=[]
    for key,entry in properties.items():
        if entry['x-kind'] not in kinds or not entry['x-owners'] or set(entry['x-owners'])-set(schema['x-services']):
            raise ValueError('invalid declaration: '+key)
        if entry['x-kind']=='integer' and not 0<=entry['minimum']<=entry['maximum']<=9007199254740991:
            raise ValueError('invalid integer bounds: '+key)
        if entry['x-kind']=='enum' and not entry.get('enum'):raise ValueError('missing enum: '+key)
        rows.append({'key':key,'kind':entry['x-kind'],'owners':entry['x-owners'],
                     'secret':entry['x-secret'],'status':entry['x-status'],
                     'aliasFor':entry.get('x-aliasFor'),'sourceReferences':refs.get(key,[])})
    return {'version':1,'schema':'config/runtime-environment.schema.json',
            'sourcePrecedence':schema['x-sourcePrecedence'],'settings':rows}

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--check',action='store_true');args=parser.parse_args()
    rendered=json.dumps(catalog(),indent=2,ensure_ascii=False)+'\n'
    if args.check:
        if not OUTPUT.exists() or OUTPUT.read_text(encoding='utf-8')!=rendered:parser.exit(1,'Runtime environment catalog differs from source/schema.\n')
    else:OUTPUT.write_bytes(rendered.encode('utf-8'))
    print('Verified runtime environment catalog: '+str(len(catalog()['settings']))+' keys')

if __name__=='__main__':main()
