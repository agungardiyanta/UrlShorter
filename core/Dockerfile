# CheckMyIP
FROM golang:1.22-alpine as build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o binary

FROM scratch as final
WORKDIR /app
COPY --from=build /app/binary/ /app/
EXPOSE 8080
ENTRYPOINT  [ "./binary" ]