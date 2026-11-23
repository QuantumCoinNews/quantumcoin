import importlib.util,json,pathlib,subprocess,sys,os,tempfile,unittest
ROOT=pathlib.Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('shared_env_under_test',ROOT/'config/runtime_environment.py')
if spec is None or spec.loader is None:raise RuntimeError('test schema loader unavailable')
shared=importlib.util.module_from_spec(spec);spec.loader.exec_module(shared)

class SharedEnvironmentTests(unittest.TestCase):
    def test_cross_language_golden_cases_and_no_mutation(self):
        cases=json.loads((ROOT/'config/runtime-environment.cases.json').read_text(encoding='utf-8'))['cases']
        self.assertGreaterEqual(len(cases),60)
        for c in cases:
            with self.subTest(case=c['name']):
                before=dict(c['environment'])
                self.assertEqual(shared.validate_environment(c['environment'],c['service']),c['errors'])
                self.assertEqual(c['environment'],before)

    def test_inventory_has_typed_owner_and_secret_classification(self):
        from runtime_environment_inventory import inventory
        from source_hygiene import source_exclusion
        for filename in ['config/runtime-environment.schema.json','config/runtime-environment.cases.json',
                         'config/runtime_environment.py','telegram_game/telegram_game/shared_environment.py']:
            self.assertFalse(source_exclusion(filename),filename)
        schema=json.loads(shared.SCHEMA_PATH.read_text(encoding='utf-8'))
        for key in inventory():
            with self.subTest(key=key):
                entry=schema['properties'][key]
                self.assertTrue(entry['x-owners']);self.assertTrue(entry['x-kind'])
        for key in ['QC_API_TOKEN','OPENAI_API_KEY','YOUTUBE_CLIENT_JSON','QC_X_STATE_ENCRYPTION_KEY',
                    'X_OAUTH2_CLIENT_ID','X_OAUTH2_CLIENT_SECRET','X_OAUTH2_TOKEN_JSON']:
            self.assertTrue(schema['properties'][key]['x-secret'])
        refs=inventory()
        for key in ['QC_DEBUG_CMD','QC_NO_PSH','QC_OVERWRITE_RUN_MINER','QC_PS_INLINE','QC_RUN_DIRECT']:
            self.assertNotIn(key,refs)
            self.assertEqual(schema['properties'][key]['x-status'],'compatibility-ignored')

    def test_unknown_keys_and_non_string_values_are_explicit_errors(self):
        self.assertEqual(shared.validate_environment({'QC_FAKE_TYPO':'1'},strict_unknown=True),{'QC_FAKE_TYPO':'unknown-setting'})
        self.assertEqual(shared.validate_environment({'QC_TOTAL_SUPPLY':12}),{'QC_TOTAL_SUPPLY':'expected-string'})

    def test_game_rejects_conflicting_config_before_creating_database(self):
        with tempfile.TemporaryDirectory(prefix='qc-shared-config-') as directory:
            database=pathlib.Path(directory)/'must-not-exist.sqlite'
            env={key:value for key,value in os.environ.items() if not key.startswith(('QC_','TELEGRAM_','QUANTUMCOIN_'))}
            env.update(PYTHONPATH=str(ROOT/'telegram_game'),QC_GAME_DB=str(database),QC_GAME_LOAD_DOTENV='0',
                       QC_GAME_MINING_MODE='mistyped-mode',TELEGRAM_BOT_TOKEN='test-only-shared-bot')
            done=subprocess.run([sys.executable,'-c','from telegram_game.runtime import app'],cwd=directory,env=env,
                                stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=60)
            self.assertNotEqual(done.returncode,0);self.assertIn(b'QC_GAME_MINING_MODE:invalid-enum',done.stdout)
            self.assertFalse(database.exists())

    def test_bot_and_identity_accept_the_same_legacy_token_aliases(self):
        for alias in ['QC_TELEGRAM_BOT_TOKEN','TELEGRAM_GAME_BOT_TOKEN']:
            with self.subTest(alias=alias),tempfile.TemporaryDirectory(prefix='qc-config-bot-alias-') as directory:
                env={key:value for key,value in os.environ.items() if not key.startswith(('QC_','TELEGRAM_','QUANTUMCOIN_'))}
                env.update(PYTHONPATH=str(ROOT/'telegram_game'),QC_GAME_DB=str(pathlib.Path(directory)/'test.sqlite'),
                           QC_GAME_LOAD_DOTENV='0',QC_API_BASE='http://127.0.0.1:1')
                env[alias]='test-only-config-bot-alias'
                code='from telegram_game.config import get_bot_token; from telegram_game.dev_api import _telegram_bot_token; assert get_bot_token()==_telegram_bot_token()=="test-only-config-bot-alias"; print("OK")'
                done=subprocess.run([sys.executable,'-c',code],cwd=directory,env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=60)
                self.assertEqual(done.returncode,0,done.stdout.decode(errors='replace'))
                self.assertEqual(done.stdout.strip(),b'OK')

    def test_alias_error_diagnostics_do_not_echo_credentials(self):
        spec=importlib.util.spec_from_file_location('game_env_redaction_test',ROOT/'telegram_game/telegram_game/shared_environment.py')
        if spec is None or spec.loader is None:raise RuntimeError('test loader unavailable')
        adapter=importlib.util.module_from_spec(spec);spec.loader.exec_module(adapter)
        with self.assertRaises(ValueError) as caught:
            adapter.require_game_environment({'TELEGRAM_BOT_TOKEN':'test-only-private-marker-a','QC_TELEGRAM_BOT_TOKEN':'test-only-private-marker-b'})
        self.assertIn('conflicting-alias',str(caught.exception))
        self.assertNotIn('test-only-private-marker',str(caught.exception))

if __name__=='__main__':unittest.main(verbosity=2)
