# Builder Image
FROM --platform=amd64 golang:1.18 as builder

# create and set working directory
RUN mkdir -p /app
WORKDIR /app

# install dependencies
ADD go.mod go.mod
ADD go.sum go.sum
RUN go mod download
# add code
ADD . .
# build the source
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go

# Final Image
FROM --platform=amd64 alpine:3.12
# add ca-certificates
RUN apk update && apk --no-cache  add ca-certificates
# set working directory
WORKDIR /app
# copy the binary from builder
COPY --from=builder /app/main ./main
# run the binary
CMD ["./main"]