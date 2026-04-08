# 1. 部署nginx 

## 安装nginx 
```
sudo apt update
sudo apt install nginx -y
```
## 启动并设置开机自启
```
sudo systemctl start nginx        # 启动Nginx服务
sudo systemctl enable nginx       # 设置开机自动启动[citation:1][citation:2]
```
## 查看nginx 版本
```
nginx -v
nginx version: nginx/1.24.0 (Ubuntu)
```

# 2. 部署Mysql

## 直接安装
```
sudo apt install mysql-server mysql-client
```
## 启动并设置开机自启
```
sudo systemctl start mysql
sudo systemctl enable mysql
```

## 查看版本
```
mysql --version
mysql  Ver 8.0.43-0ubuntu0.24.04.2 for Linux on x86_64 ((Ubuntu))
```

## 配置MySQL允许外部连接
```
/etc/mysql/mysql.conf.d/mysqld.cnf
```
```
bind-address            = 0.0.0.0
```

## 创建用户并授予所有权限
```
mysql -u root -p
```
可以一次性执行以下所有命令：
```
-- 登录MySQL后执行以下命令
-- 1. 创建用户
CREATE USER 'user'@'%' IDENTIFIED BY 'MySQL@123456!';

-- 2. 授予所有权限
GRANT ALL PRIVILEGES ON *.* TO 'user'@'%' WITH GRANT OPTION;

-- 3. 创建本地连接用户（可选）
CREATE USER 'user'@'localhost' IDENTIFIED BY 'MySQL@123456!';
GRANT ALL PRIVILEGES ON *.* TO 'user'@'localhost' WITH GRANT OPTION;

-- 4. 刷新权限
FLUSH PRIVILEGES;

-- 5. 查看用户权限
SHOW GRANTS FOR 'user'@'%';
SHOW GRANTS FOR 'user'@'localhost';

-- 6. 查看所有用户
SELECT user, host FROM mysql.user;
```

## 重启MySQL服务
```
sudo systemctl restart mysql
```

# 3. 部署 Redis 

## 直接安装
```
sudo apt install redis-server
```

## 配置 Redis
```
vim /etc/redis/redis.conf
```
```
# 允许所有连接
bind 0.0.0.0 -::1
protected-mode no

# 密码认证
requirepass pass123

# 性能设置
maxmemory 1gb #最大内存
maxmemory-policy volatile-lru #只从设置了过期时间的键中淘汰
appendonly no #数据持久化保障，防止数据丢失
```

## 启动和启用 Redis 服务
```
# 启动 Redis 服务
sudo systemctl start redis-server

# 设置开机自启
sudo systemctl enable redis-server

# 检查服务状态
sudo systemctl status redis-server
```

# 4. 部署docker
## 使用阿里云镜像安装
```
# 使用阿里云的安装脚本
curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | sudo apt-key add -

# 添加阿里云 Docker CE 源
sudo add-apt-repository "deb [arch=amd64] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable"

# 安装 Docker
sudo apt update
sudo apt install docker-ce docker-ce-cli containerd.io
```