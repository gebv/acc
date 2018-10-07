
setup-schema:
	# TODO: setup PGPASSWORD
	psql -U acca -q -h 127.0.0.1 -d acca -U acca -v ON_ERROR_STOP=1 -f ./schema.sql
.PHONY: setup-schema

setup-functions:
	# TODO: setup PGPASSWORD
	psql -U acca -q -h 127.0.0.1 -d acca -U acca -v ON_ERROR_STOP=1 -f ./functions.sql
.PHONY: setup-functions

setup: setup-schema setup-functions
.PHONY: setup
