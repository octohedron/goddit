FROM golang:1.7.3
WORKDIR /go/src/github.com/octohedron/goddit
COPY . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o goddit .

FROM alpine:latest  
RUN apk add --no-cache ca-certificates 
WORKDIR /root/
COPY --from=0 /go/src/github.com/octohedron/goddit .


CMD ["./goddit"]