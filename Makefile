.PHONY: proto

proto:
	cd api/grpc && buf generate
