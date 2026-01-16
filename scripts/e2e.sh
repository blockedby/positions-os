#!/bin/bash
# scripts/e2e.sh - Run E2E tests with ephemeral infrastructure

set -e

echo "Starting E2E tests with ephemeral infrastructure..."

# Build and run, exit when e2e-runner completes
docker compose -f docker-compose.e2e.yml up \
  --build \
  --abort-on-container-exit \
  --exit-code-from e2e-runner

EXIT_CODE=$?

echo "Cleaning up containers..."
docker compose -f docker-compose.e2e.yml down --volumes --remove-orphans

if [ $EXIT_CODE -eq 0 ]; then
  echo "E2E tests passed!"
else
  echo "E2E tests failed with exit code $EXIT_CODE"
  echo "Check frontend/test-results/ for screenshots"
fi

exit $EXIT_CODE
