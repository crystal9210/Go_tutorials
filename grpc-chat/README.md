【参照】
https://github.com/rodaine/grpc-chat/blob/main/server.go

【commands】
①protoc --proto_path=protos --go_out=protos --go_opt=paths=source_relative --go-grpc_out=protos --go-grpc_opt=paths=source_relative protos/chat.proto
→you can go file from .proto
