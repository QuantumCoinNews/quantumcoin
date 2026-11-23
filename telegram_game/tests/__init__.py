"""Safe standard unittest discovery; scenarios execute in isolated children."""


def load_tests(loader, tests, pattern):
    from isolated_contracts import runtime_suite
    return runtime_suite()
