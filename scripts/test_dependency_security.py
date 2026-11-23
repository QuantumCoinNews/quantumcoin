import unittest

from verify_dependency_security import go_findings, npm_findings, python_findings


class SecurityResultAcceptanceTest(unittest.TestCase):
    def test_go_zero_exit_json_can_still_contain_findings(self):
        result = go_findings('{"config":{}}\n{"finding":{"osv":"GO-fixture","trace":[]}}')
        self.assertEqual(result[0]["osv"], "GO-fixture")

    def test_go_clean_result_and_first_download_are_understood(self):
        self.assertEqual(go_findings('{"config":{}}\n{"progress":{}}\ngo: downloading example.test/tool v1.0.0\n'), [])

    def test_go_missing_and_malformed_results_cannot_pass(self):
        for value in ("", '{"progress":{}}', '{"config":{}}{', '{"config":{}}{"finding":{}}',
                      '{"config":{}}{"error":"unavailable"}'):
            with self.subTest(value=value), self.assertRaises(ValueError):
                go_findings(value)

    def test_npm_requires_complete_numeric_counts(self):
        clean = '{"metadata":{"vulnerabilities":{"info":0,"low":0,"moderate":0,"high":0,"critical":0,"total":0}}}'
        self.assertEqual(npm_findings(clean)["total"], 0)
        self.assertEqual(npm_findings(clean.replace('"high":0', '"high":1').replace('"total":0', '"total":1'))["total"], 1)
        for value in ('{}', '{"error":{"code":"fixture"}}', clean.replace('"low":0', '"low":false')):
            with self.subTest(value=value), self.assertRaises(ValueError):
                npm_findings(value)

    def test_python_findings_and_skipped_packages_are_not_clean(self):
        self.assertEqual(python_findings({"dependencies": [{"name": "fixture", "vulns": []}]}), [])
        result = python_findings({"dependencies": [{"name": "fixture", "vulns": [{"id": "fixture-CVE"}]}]})
        self.assertEqual(result[0]["package"], "fixture")
        for value in ({}, {"dependencies": []}, {"dependencies": [{"name": "fixture", "skip_reason": "unavailable", "vulns": []}]}):
            with self.subTest(value=value), self.assertRaises(ValueError):
                python_findings(value)


if __name__ == "__main__":
    unittest.main()
