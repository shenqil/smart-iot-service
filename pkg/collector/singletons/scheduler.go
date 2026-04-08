package singletons

import (
	"fmt"
	"rbac-admin/pkg/logger"
	"sync"
	"time"
)

// IntervalTask 间隔任务
type IntervalTask struct {
	Name     uint32
	Interval time.Duration
	StopChan chan struct{}
	Task     func()
}

// Scheduler 调度器单例
type Scheduler struct {
	mu    sync.Mutex
	tasks map[uint32]*IntervalTask
}

var (
	schedulerInstance *Scheduler
	schedulerOnce     sync.Once
)

// GetScheduler 获取调度器单例
func GetScheduler() *Scheduler {
	schedulerOnce.Do(func() {
		schedulerInstance = &Scheduler{
			tasks: make(map[uint32]*IntervalTask),
		}
	})
	return schedulerInstance
}

// StartTask 启动任务
func (s *Scheduler) StartTask(taskID uint32, intervalMinutes int, task func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if intervalMinutes <= 0 {
		return fmt.Errorf("间隔分钟数必须 > 0")
	}

	interval := time.Duration(intervalMinutes) * time.Minute
	stopChan := make(chan struct{})

	// 若已有任务，先停止
	if oldTask, ok := s.tasks[taskID]; ok {
		close(oldTask.StopChan)
		delete(s.tasks, taskID)
	}

	// 启动任务 goroutine
	go func(taskID uint32, interval time.Duration, stopChan chan struct{}) {
		task()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				logger.Logger.Info(fmt.Sprintf("任务 [%d] 已停止", taskID))
				return
			case <-ticker.C:
				logger.Logger.Debug(fmt.Sprintf("执行任务 [%d] at %s", taskID, time.Now().Format("15:04:05")))
				task()
			}
		}
	}(taskID, interval, stopChan)

	// 保存任务
	s.tasks[taskID] = &IntervalTask{
		Name:     taskID,
		Interval: interval,
		StopChan: stopChan,
		Task:     task,
	}

	logger.Logger.Info(fmt.Sprintf("✅ 启动任务 [%d]，间隔 %v", taskID, interval))
	return nil
}

// StopTask 停止任务
func (s *Scheduler) StopTask(taskID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, ok := s.tasks[taskID]; ok {
		close(task.StopChan)
		delete(s.tasks, taskID)
		logger.Logger.Info(fmt.Sprintf("🗑️ 已删除任务 [%d]", taskID))
	}
}

// StopAllTasks 停止所有任务
func (s *Scheduler) StopAllTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for taskID, task := range s.tasks {
		close(task.StopChan)
		delete(s.tasks, taskID)
		logger.Logger.Info(fmt.Sprintf("🗑️ 已删除任务 [%d]", taskID))
	}
}

// ListTasks 列出所有任务
func (s *Scheduler) ListTasks() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var list []uint32
	for taskID := range s.tasks {
		list = append(list, taskID)
	}
	return list
}
