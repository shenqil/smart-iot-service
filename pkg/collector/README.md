# Collector 模块完全重构说明

## 架构概述

完全重构collector模块，采用全新的模块化架构，移除向后兼容层：

```
pkg/collector/
├── core/                    # 核心模块
│   ├── types.go            # 类型定义和接口
│   └── collector.go        # 核心采集器实现
├── services/               # 采集服务模块
│   ├── connection_manager.go # 连接管理器
│   └── iot_service.go      # IoT协议服务
├── singletons/             # 单例模式模块
│   ├── scheduler.go        # 调度器单例
│   └── event_publisher.go  # 事件发布器单例
├── storage/                # 存储模块
│   └── storage.go          # 存储接口实现
├── protocols/              # 协议处理模块
│   └── iot_protocol.go     # IoT协议处理器
└── main.go                 # 主入口文件（向后兼容）
```

## 模块说明

### 1. 核心模块 (core/)
- **types.go**: 定义所有接口和数据结构
- **collector.go**: 核心采集器实现，负责服务注册、启动停止等

### 2. 采集服务模块 (services/)
- **connection_manager.go**: 连接管理器，管理设备连接
- **active_report_service.go**: 主动上报服务，实现core.ServiceInterface接口
- **control_service.go**: 控制服务，实现core.ServiceInterface接口
- **iot_service.go**: IoT协议服务，实现core.ServiceInterface接口

### 3. 单例模式模块 (singletons/)
- **scheduler.go**: 调度器单例，管理定时任务
- **event_publisher.go**: 事件发布器单例，处理事件发布订阅

### 4. 存储模块 (storage/)
- **storage.go**: 存储接口实现，提供数据存储API

### 5. 协议处理模块 (protocols/)
- **iot_protocol.go**: IoT协议处理器，处理协议解析

## 主要特性

### 1. 完全重构
- 移除所有向后兼容层
- 全新的模块化架构设计
- 更清晰的代码结构

### 2. 正确接口设计
- 核心模块定义接口，服务模块实现接口
- 三个独立服务，都实现core.ServiceInterface接口
- 支持接口多态，便于扩展和维护

### 3. 模块化设计
- 每个模块职责单一，易于维护
- 模块间通过接口通信，降低耦合
- 避免循环导入问题

### 4. 单例模式
- 调度器和事件发布器采用单例模式
- 整个模块只需要一个实例

### 5. 连接和协议分离
- 连接管理独立于协议处理
- 协议处理器可复用

### 6. 三个独立服务
- **主动上报服务**: 处理设备主动上报的数据
- **控制服务**: 处理设备控制命令和定时任务
- **IoT协议服务**: 处理IoT协议通信
- **统一接口**: 三个服务都实现相同的服务接口

## 使用方式

### 创建采集器
```go
// 直接创建采集器，自动初始化三个服务
collector := collector.NewCollector(db, influxClient, deviceCache, pubsubRedis)
```

### 启动采集器
```go
err := collector.Start()
if err != nil {
    log.Fatal(err)
}
```

### 停止采集器
```go
collector.Stop()
```

### 设备控制
```go
// 断开设备连接
collector.DisconnectedDevice(deviceID)

// 控制继电器
err := collector.RelaySwitch(deviceID, relayIndex, isOpen)

// 设置读取间隔
err := collector.SetDeviceReadInterval(deviceID, intervalMinutes)

// 获取连接的设备
devices := collector.GetConnectedDevices()

// 检查设备是否连接
connected := collector.IsDeviceConnected(deviceID)
```

### 服务管理
```go
// 获取服务
service1, exists1 := collector.GetService("主动上报服务")
service2, exists2 := collector.GetService("控制服务")
service3, exists3 := collector.GetService("IoT协议服务")

// 注意：三个服务在创建时已自动注册
// 每个服务都实现了统一的服务接口
```

## 扩展指南

### 添加新的协议服务
1. 在 `services/` 目录下创建新的服务文件
2. 实现 `ServiceInterface` 接口
3. 在 `main.go` 中注册服务

### 添加新的存储后端
1. 在 `storage/` 目录下扩展存储实现
2. 实现 `StorageInterface` 接口
3. 更新存储单例

### 添加新的事件类型
1. 在 `core/types.go` 中定义新的事件类型
2. 在相应服务中发布事件
3. 在事件处理器中处理新事件

## 优势

1. **易于维护**: 模块化设计，职责清晰
2. **易于扩展**: 接口设计，支持插件化
3. **高性能**: 单例模式，减少资源消耗
4. **完全重构**: 全新的架构设计
5. **测试友好**: 模块独立，易于单元测试
