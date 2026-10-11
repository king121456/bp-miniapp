FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /bp-server .

FROM gcr.io/distroless/static
COPY --from=build /bp-server /bp-server
EXPOSE 8080
ENTRYPOINT ["/bp-server"]
