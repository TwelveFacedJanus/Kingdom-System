#!/bin/bash

protoc -I=../Libraries/proto \
  --go_out=../Services/Mikhail/gen/proto --go_opt=paths=source_relative \
  --go-grpc_out=../Services/Mikhail/gen/proto --go-grpc_opt=paths=source_relative \
  ../Libraries/proto/Authenticate.proto
