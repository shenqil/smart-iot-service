package controlv2

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"rbac-admin/pkg/collector/singletons"
	"rbac-admin/pkg/logger"
	"sync"
	"time"

	"github.com/panjf2000/gnet/v2"
)

type ControlConnContextV2 struct {
	mu             sync.Mutex    
	deviceID       uint32        
	timeoutTimer   *time.Timer   
	buffer         *bytes.Buffer 
	connectionID   string        
	connectionTime int64         
}

type ControlServiceV2 struct {
	gnet.BuiltinEventEngine
	eng         gnet.Engine
	addr        string
	ctx         context.Context
	cancel      context.CancelFunc
	name        string
	logger      *slog.Logger
	deviceConns sync.Map 
	scheduler   *singletons.Scheduler
}

func NewControlServiceV2(addr string) *ControlServiceV2 {
	ctx, cancel := context.WithCancel(context.Background())
	return &ControlServiceV2{
		addr:      addr,
		ctx:       ctx,
		cancel:    cancel,
		name:      "控制服务V2",
		logger:    logger.NewServiceLogger("control_service_v2"),
		scheduler: singletons.GetScheduler(),
	}
}

func (s *ControlServiceV2) GetName() string {
	return s.name
}

func (s *ControlServiceV2) Start() error {
	s.logger.Info(fmt.Sprintf("启动 %s，监听地址: %s", s.name, s.addr))
	if err := gnet.Run(s, s.addr, gnet.WithMulticore(true)); err != nil {
		s.logger.Error(fmt.Sprintf("启动 %s 失败: %v", s.name, err))
		return err
	}
	s.logger.Info(fmt.Sprintf("%s 启动成功", s.name))
	return nil
}

func (s *ControlServiceV2) Stop() error {
	s.logger.Info(fmt.Sprintf("停止 %s", s.name))
	s.cancel()
	if s.scheduler != nil {
		s.scheduler.StopAllTasks()
	}
	s.deviceConns.Range(func(key, value interface{}) bool {
		if conn, ok := value.(gnet.Conn); ok {
			conn.Close()
		}
		return true
	})
	s.logger.Info(fmt.Sprintf("%s 已停止", s.name))
	return nil
}

func (s *ControlServiceV2) OnBoot(eng gnet.Engine) gnet.Action {
	s.eng = eng
	s.logger.Info(fmt.Sprintf("%s 服务启动，监听地址: %s", s.name, s.addr))
	return gnet.None
}

func (s *ControlServiceV2) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	ctx := &ControlConnContextV2{
		deviceID:       0,
		buffer:         bytes.NewBuffer(nil),
		connectionID:   c.RemoteAddr().String(),
		connectionTime: time.Now().Unix(),
	}

	ctx.mu.Lock()
	ctx.timeoutTimer = time.AfterFunc(30*time.Second, func() {
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		if ctx.deviceID == 0 {
			s.logger.Warn(fmt.Sprintf("%s 连接 %s 在30秒内未收到设备ID，主动关闭 (防死链保护)", s.name, c.RemoteAddr().String()))
			c.Close()
		}
	})
	ctx.mu.Unlock()

	c.SetContext(ctx)
	s.logger.Info(fmt.Sprintf("%s 客户端连接: %s", s.name, c.RemoteAddr().String()))

	idRequestCmd := s.BuildReadIDCommand()
	return idRequestCmd, gnet.None
}

func (s *ControlServiceV2) OnClose(c gnet.Conn, err error) gnet.Action {
	addr := c.RemoteAddr().String()

	if ctxInterface := c.Context(); ctxInterface != nil {
		if ctx, ok := ctxInterface.(*ControlConnContextV2); ok {
			ctx.mu.Lock()
			if ctx.timeoutTimer != nil {
				ctx.timeoutTimer.Stop()
			}
			deviceID := ctx.deviceID
			ctx.mu.Unlock()

			if deviceID != 0 {
				if existingConnInterface, exists := s.deviceConns.Load(deviceID); exists {
					if existingConn, ok := existingConnInterface.(gnet.Conn); ok {
						if existingConn == c {
							s.deviceConns.Delete(deviceID)
							s.scheduler.StopTask(deviceID)
							s.publishEvent(singletons.EventStatus, &singletons.DeviceStatus{
								ID:           deviceID,
								Online:       0,
								Time:         time.Now(),
								ConnectionID: addr,
							})
							s.logger.Info(fmt.Sprintf("%s 清理设备映射: %d", s.name, deviceID))
						}
					}
				}
			}
		}
	}
	s.logger.Info(fmt.Sprintf("%s 客户端断开: %s, 错误: %v", s.name, addr, err))
	return gnet.None
}

func (s *ControlServiceV2) OnTraffic(c gnet.Conn) gnet.Action {
	ctx, ok := c.Context().(*ControlConnContextV2)
	if !ok || ctx == nil {
		return gnet.Close
	}

	data, err := c.Next(-1)
	if err != nil {
		s.logger.Error(fmt.Sprintf("%s 读取数据错误: %v", s.name, err))
		return gnet.Close
	}
	if len(data) == 0 {
		return gnet.None
	}

	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.buffer.Len()+len(data) > 4096 {
		s.logger.Warn(fmt.Sprintf("%s 连接 %s 缓冲区溢出，强制重置", s.name, c.RemoteAddr().String()))
		ctx.buffer.Reset()
	}
	ctx.buffer.Write(data)

	for ctx.buffer.Len() > 0 {
		parsedLen := s.tryParsePacket(ctx.buffer.Bytes(), ctx, c)
		if parsedLen > 0 {
			ctx.buffer.Next(parsedLen)
		} else if parsedLen == 0 {
			break
		} else {
			ctx.buffer.Next(1)
		}
	}
	return gnet.None
}
