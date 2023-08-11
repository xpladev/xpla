#!/usr/bin/env bash

set -eo pipefail

mkdir -p ./tmp-swagger-gen

# move the vendor folder to a temp dir so that go list works properly
temp_dir="f29ea6aa861dc4b083e8e48f67cce"
if [ -d vendor ]; then
  mv ./vendor ./$temp_dir
fi

# Get the path of the cosmos-sdk repo from go/pkg/mod
gogo_proto_dir=$(go list -f '{{ .Dir }}' -m github.com/gogo/protobuf)
google_api_dir=$(go list -f '{{ .Dir }}' -m github.com/grpc-ecosystem/grpc-gateway)
tendermint_dir=$(go list -f '{{ .Dir }}' -m github.com/tendermint/tendermint)
cosmos_proto_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-proto)
cosmos_sdk_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-sdk)
wasm_dir=$(go list -f '{{ .Dir }}' -m github.com/CosmWasm/wasmd)
ibc_dir=$(go list -f '{{ .Dir }}' -m github.com/cosmos/ibc-go/v4)
ica=$(go list -f '{{ .Dir }}' -m github.com/cosmos/interchain-accounts)
ethermint_dir=$(go list -f '{{ .Dir }}' -m github.com/evmos/ethermint)
pfm=$(go list -f '{{ .Dir }}' -m github.com/strangelove-ventures/packet-forward-middleware/v4)
xpla_dir=$(go list -f '{{ .Dir }}' -m github.com/xpladev/xpla)

# move the vendor folder back to ./vendor
if [ -d $temp_dir ]; then
  mv ./$temp_dir ./vendor
fi

proto_dirs=$(find $tendermint_dir/proto $cosmos_sdk_dir/proto $wasm_dir/proto $ibc_dir/proto $ethermint_dir/proto $ica/proto $pfm/proto $xpla_dir/proto -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  # generate swagger files (filter query files)
  query_file=$(find "${dir}" -maxdepth 1 \( -name 'query.proto' -o -name 'service.proto' \))
  if [[ ! -z "$query_file" ]]; then
    protoc  \
    -I "$tendermint_dir/proto" \
    -I "$cosmos_sdk_dir/proto" \
    -I "$ibc_dir/proto" \
    -I "$ethermint_dir/proto" \
    -I "$xpla_dir/proto" \
    -I "$ica/proto" \
    -I "$pfm/proto" \
    -I "$wasm_dir/proto" \
    -I "$gogo_proto_dir" \
    -I "$google_api_dir/third_party/googleapis" \
    -I "$cosmos_proto_dir/proto" \
    -I "third_party/proto" \
      "$query_file" \
    --swagger_out ./tmp-swagger-gen \
    --swagger_opt logtostderr=true \
    --swagger_opt fqn_for_swagger_name=true \
    --swagger_opt simple_operation_ids=true
  fi
done


# Fix circular definition in cosmos/tx/v1beta1/service.swagger.json
jq 'del(.definitions["cosmos.tx.v1beta1.ModeInfo.Multi"].properties.mode_infos.items["$ref"])' ./tmp-swagger-gen/cosmos/tx/v1beta1/service.swagger.json > ./tmp-swagger-gen/cosmos/tx/v1beta1/fixed-service.swagger.json
rm ./tmp-swagger-gen/cosmos/tx/v1beta1/service.swagger.json

# Tag everything as "gRPC Gateway API"
perl -i -pe 's/"(Query|Service)"/"gRPC Gateway API"/' $(find ./tmp-swagger-gen -name '*.swagger.json' -print0 | xargs -0)

# Convert all *.swagger.json files into a single folder _all
files=$(find ./tmp-swagger-gen -name '*.swagger.json' -print0 | xargs -0)
mkdir -p ./tmp-swagger-gen/_all
counter=0
for f in $files; do
  echo "[+] $f"

  # check gaia first before cosmos
  if [[ "$f" =~ "router" ]]; then
    cp $f ./tmp-swagger-gen/_all/pfm-$counter.json
  elif [[ "$f" =~ "cosmwasm" ]]; then
    cp $f ./tmp-swagger-gen/_all/cosmwasm-$counter.json
  elif [[ "$f" =~ "ethermint" ]]; then
    cp $f ./tmp-swagger-gen/_all/ethermint-$counter.json
  elif [[ "$f" =~ "xpla" ]]; then
    cp $f ./tmp-swagger-gen/_all/xpla-$counter.json
  elif [[ "$f" =~ "cosmos" ]]; then
    cp $f ./tmp-swagger-gen/_all/cosmos-$counter.json
  elif [[ "$f" =~ "pfm" ]]; then
    cp $f ./tmp-swagger-gen/_all/pfm-$counter.json
  else
    cp $f ./tmp-swagger-gen/_all/other-$counter.json
  fi
  ((counter++))
done

# merges all the above into FINAL.json
python3 ./scripts/merge_protoc.py

# Makes a swagger temp file with reference pointers
swagger-combine ./tmp-swagger-gen/_all/FINAL.json -o ./docs/_tmp_swagger.yaml -f yaml --continueOnConflictingPaths --includeDefinitions

# extends out the *ref instances to their full value
swagger-merger -i ./docs/_tmp_swagger.yaml -o ./docs/swagger.yaml

# Derive openapi from swagger docs
swagger2openapi --patch ./docs/swagger.yaml --outfile ./docs/static/openapi.yml --yaml  

# Sort
yaml-sort --input ./docs/static/openapi.yml

# clean swagger tmp files
rm ./docs/_tmp_swagger.yaml
rm -rf ./tmp-swagger-gen