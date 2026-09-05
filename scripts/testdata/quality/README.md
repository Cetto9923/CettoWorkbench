# Quality Gate Test Fixtures

This directory contains test data and fixtures for validating quality gate scripts (`scripts/test-quality-gates.sh`).
These fixtures ensure:
1. Scripts correctly exit 0 when both current diagnostics and baselines are empty.
2. Scripts do not fail due to `set -euo pipefail` on empty grep pipelines.
3. Over-limit files are detected even when baseline files are 0 bytes or contain only comments.
4. Tool failures (non-zero exits without diagnostics) are not swallowed.
