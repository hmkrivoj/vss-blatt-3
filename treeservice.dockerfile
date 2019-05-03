FROM obraun/vss-protoactor-jenkins as builder
COPY . /app
WORKDIR /app
RUN go get github.com/urfave/cli
RUN go build -o treeservice/main treeservice/main.go

FROM iron/go
COPY --from=builder /app/treeservice/main /app/treeservice
EXPOSE 8090
ENTRYPOINT ["/app/treeservice"]
