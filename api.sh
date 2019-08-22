#!/bin/bash -x

# ./api.sh '{"key":"curr"}' "api.Accounts/CreateCurrency"
# ./api.sh '{"key":"curr"}' "api.Accounts/GetCurrency"
# ./api.sh '{}' "api.Updates/GetUpdate"

grpcurl \
  -rpc-header 'x-request-id: api-sh-test-123' \
  -rpc-header 'access-token: dcb01ad8bc58fb43d93eab37b08f3ba6b0011ceef64cb24abd29db0934e8bc6e' \
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
  127.0.0.1:10011 \
  $2
