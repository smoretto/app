FROM golang:1.22.3-alpine as build

WORKDIR /src

COPY go.mod .
COPY go.sum .

RUN go mod download
RUN go mod verify

COPY . .

ENV CGO_ENABLED=0

ARG version=undef

RUN go build -ldflags "-X main.version=$version" -o app cmd/app/main.go
FROM scratch AS app

COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/group /etc/group
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

USER nobody:nobody

WORKDIR /app
EXPOSE 8080

COPY --from=build /src/app /app

ENTRYPOINT [ "/app/app" ]
