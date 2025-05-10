dev:
	go build -o cloud_file_storage cmd/main.go
	./cloud_file_storage

build:
	go build -o cloud_file_storage cmd/main.go

m_up:
	goose up

m_down:
	goose down

m_create:
	goose create $(name) sql
