GO        = go
PROTOC    = protoc
PROTO_DIR = proto

gen-protobuf:
	$(PROTOC) --proto_path=$(PROTO_DIR) --go_out=plugins=grpc:$(PROTO_DIR) $(PROTO_DIR)/*.protoe

test:
	$(GO) test -v ./...