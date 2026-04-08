# 更新 swag

```
go generate .
```
或者
```
swag init --parseDependency --parseInternal
```

# 更新依赖注入

```
go generate ./internal/app
```
或者
```
wire ./internal/app
go run github.com/google/wire/cmd/wire ./internal/app
```

# 运行

```
go run .\main.go
```

# 编译

```
go build .\main.go
```

# docker 编译

```
docker build -t fssq/smart-iot-admin:v1.0.0 .
docker push fssq/smart-iot-admin:v1.0.0
```

## 阿里云

```
docker login --username=q15623509401 crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com
docker build -t crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/smart-iot-admin:latest .

docker tag [ImageId] crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/smart-iot-admin:[镜像版本号]
docker push crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/smart-iot-admin:latest

docker-compose down --rmi all
docker pull crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/smart-iot-admin:[镜像版本号]
```

## mac 编译
```
docker build --platform linux/amd64  -t crpi-14aq4ypr9y9smv5q.cn-hangzhou.personal.cr.aliyuncs.com/fssq/smart-iot-admin:latest .
```

# docker 运行

```
docker compose stop
docker compose down --rmi all
docker compose up -d --build

docker compose up -d --build
docker compose logs -f smart-iot-admin
tail -f ~/gin-admin/logs/gin-admin.log
```
