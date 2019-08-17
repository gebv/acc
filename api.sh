#!/bin/bash -x

# ./api.sh '{"key":"curr"}' "api.Accounts/CreateCurrency"
# ./api.sh '{"key":"curr"}' "api.Accounts/GetCurrency"
# ./api.sh '{}' "api.Updates/GetUpdate"

grpcurl \
  -rpc-header 'x-request-id: api-sh-test-123' \
  -rpc-header 'access-token: cfb25eb22addb7d7edb743388a0f6b406e3fea368f2dd742c18a6a816791d74e' \
  -plaintext \
  -import-path ./ \
  -proto ./api/accounts.proto \
  -proto ./api/invoice.proto \
  -proto ./api/update.proto \
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
