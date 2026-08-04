FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /orders-api main.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /orders-api /orders-api
EXPOSE 8080
ENTRYPOINT ["/orders-api"]
