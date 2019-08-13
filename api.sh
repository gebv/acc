#!/bin/bash -x

# ./api.sh '{"key":"curr"}' "api.Accounts/CreateCurrency"

grpcurl \
  -rpc-header 'x-request-id: api-sh-test-123' \
  -rpc-header 'device-id: api.sh' \
  -plaintext \
  -import-path ./ \
  -proto ./api/accounts.proto \
  -proto ./api/invoice.proto \
  -import-path ./vendor \
  -proto github.com/mwitkow/go-proto-validators/validator.proto \
  -proto github.com/gogo/protobuf/gogoproto/gogo.proto \
  -proto google/protobuf/timestamp.proto \
  -proto google/protobuf/empty.proto \
  -proto google/protobuf/wrappers.proto \
  -v \
  -d $1 \
  127.0.0.1:10001 \
  $2

