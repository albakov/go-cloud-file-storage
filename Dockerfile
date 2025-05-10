FROM golang:1.24.3 AS build-stage

ARG CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# -tags=viper_bind_struct allow viper read env vars from docker
RUN go build -o /app/server -tags=viper_bind_struct cmd/main.go

FROM scratch
COPY --from=build-stage /app/server /server
CMD ["/server"]