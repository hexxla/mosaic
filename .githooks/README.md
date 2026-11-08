# Optional Git hook

The pre-commit hook scans staged changes for secrets. Install the pinned scanner
with `scripts/tool.sh install-gitleaks` and add Go’s bin directory to your PATH.
Run the same check directly with `scripts/check-secrets.sh --staged`.

To enable the repository hook in this clone, run:

```sh
git config --local core.hooksPath .githooks
```

Enable it only if you want this clone to use the repository hook directory.
CI checks the pushed commits independently.
