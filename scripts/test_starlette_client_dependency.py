"""Test the installed client dependency without hiding compatibility warnings."""
import importlib
import unittest
import warnings


class StarletteClientDependencyTest(unittest.TestCase):
    def test_testclient_uses_locked_httpx2_without_deprecation(self):
        with warnings.catch_warnings():
            warnings.simplefilter("error")
            # Discovery may already have imported TestClient; reload explicitly
            # so this test still exercises its optional dependency selection.
            import starlette.testclient
            module = importlib.reload(starlette.testclient)
            from starlette.applications import Starlette
            from starlette.responses import JSONResponse
            from starlette.routing import Route
            async def health(_request):
                return JSONResponse({"ok": True})
            self.assertEqual(module.httpx.__name__, "httpx2")
            with module.TestClient(Starlette(routes=[Route("/health", health)])) as client:
                self.assertEqual(client.get("/health").json(), {"ok": True})


if __name__ == "__main__":
    unittest.main()
