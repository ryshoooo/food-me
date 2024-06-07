FROM docker.io/library/golang:1.22-alpine as builder

# Setup
RUN mkdir -p /go/src/github.com/ryshoooo/food-me
WORKDIR /go/src/github.com/ryshoooo/food-me

# Add libraries
RUN apk add --no-cache git

# Copy & build
ADD . /go/src/github.com/ryshoooo/food-me
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix nocgo -o /foodme github.com/ryshoooo/food-me/cmd

# Copy into scratch container
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /foodme ./
ENTRYPOINT ["/foodme"]
