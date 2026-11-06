# Shim: the real task definitions live in ./justfile.
# `make <target>` forwards to `just <target>`; prefer `just` directly.
.PHONY: default run-build run-dev run-migrate run-seed test tidy

default run-build run-dev run-migrate run-seed test tidy:
	@just $@
