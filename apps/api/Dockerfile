FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY apps/api ./apps/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/urlshorter-api ./apps/api/cmd

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/urlshorter-api /app/urlshorter-api
USER app

EXPOSE 8080
ENTRYPOINT ["/app/urlshorter-api"]
