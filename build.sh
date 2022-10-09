#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

SET CGO_ENABLED=0
SET GOOS=linux
SET GOARCH=amd64
go build -o main
ssh root@192.168.101.21 "cd /jhmk/jhdcp/container/jhdcp-om/;docker-compose down"
scp main root@192.168.101.21:/jhmk/jhdcp/container/jhdcp-om/bin/
ssh root@192.168.101.21 "cd /jhmk/jhdcp/container/jhdcp-om/;docker-compose up -d"


