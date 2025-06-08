# To regenerate proto:

poetry run python -m grpc_tools.protoc -I/Users/twelvefaced/Documents/Kingdom-System/Libraries/proto --python_out=proto/ --grpc_python_out=proto/ /Users/twelvefaced/Documents/Kingdom-System/Libraries/proto/Authenticate.proto