FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bp-server ./cmd/server

FROM gcr.io/distroless/static
COPY --from=build /bp-server /bp-server
EXPOSE 8080
ENTRYPOINT ["/bp-server"]
