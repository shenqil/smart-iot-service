FROM crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/golang:v1.24.4 AS builder

WORKDIR /app

RUN go env -w GOPROXY=https://goproxy.cn,direct

# 先只复制依赖文件，利用Docker缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制其余文件
COPY . .

RUN GOOS=linux CGO_ENABLED=1 GOARCH=amd64 go build -o /gin-admin

FROM crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/debian:12

# 从builder阶段复制必要的文件
COPY --from=builder /gin-admin /gin-admin

EXPOSE 10088 9000 9999 9898 9998

# 设置启动命令
ENTRYPOINT ["/gin-admin"]