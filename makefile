.PHONY: serve tidy test

serve:
	go run cmd/api/main.go
tidy:
	go mod tidy 
test:
	go run cmd/test/main.go