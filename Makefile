.PHONY: genproto

genproto:
	protoc -Iproto --go_out=proto --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative  --validate_out="lang=go,paths=source_relative:proto" .\proto\todo\v2\*.proto