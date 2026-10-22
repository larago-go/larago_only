#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# Use an explicit workspace Go installation when the system `go` is another program.
if [[ -f /workspace/.larago-tools/activate.sh ]]; then source /workspace/.larago-tools/activate.sh; fi
if [[ ! -f backend/.env ]]; then cp backend/.env.example backend/.env; fi
set -a
source backend/.env
set +a
cd backend
go run -mod=readonly .
