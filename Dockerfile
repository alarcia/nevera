FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Tests run inside the build: a failure stops the deploy before anything is replaced.
RUN CGO_ENABLED=0 go test ./... \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /nevera .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /nevera /nevera
EXPOSE 8090
ENTRYPOINT ["/nevera", "serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]
