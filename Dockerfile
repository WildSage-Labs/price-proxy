FROM golang:alpine AS build
RUN apk --no-cache add ca-certificates
RUN addgroup -S myapp && adduser -S -u 10000 -g myapp myapp
WORKDIR /src
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY ./ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -installsuffix 'static' -o /app

# STAGE 2: build the container to run
FROM scratch AS final
LABEL maintainer="gbaeke"
COPY --from=build  /app /pp
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/pp"]