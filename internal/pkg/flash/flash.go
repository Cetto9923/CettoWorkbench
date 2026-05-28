// =============================================================================
// 文件: internal/pkg/flash/flash.go
// 模块: 基础设施
// 类型: infra
// 职责: 封装 Flash 消息结构与读写操作。
// 依赖: internal/pkg/logger
// =============================================================================

package flash

import (
	"encoding/gob"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"

	"workbench/internal/pkg/logger"
)

// Level 表示 Flash 消息级别。
type Level string

const (
	LevelSuccess Level = "success"
	LevelError   Level = "error"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
)

const sessionKey = "_flash"

var defaultSessionMgr *scs.SessionManager

// Message 表示一个可渲染的 Flash 消息。
type Message struct {
	Level Level
	Text  string
}

// SetDefault 注册默认 SessionManager。
func SetDefault(mgr *scs.SessionManager) {
	gob.Register(Message{})
	defaultSessionMgr = mgr
}

// Success 写入成功提示。
func Success(c *gin.Context, msg string) { put(c, LevelSuccess, msg) }

// Error 写入错误提示。
func Error(c *gin.Context, msg string) { put(c, LevelError, msg) }

// Info 写入信息提示。
func Info(c *gin.Context, msg string) { put(c, LevelInfo, msg) }

// Warning 写入警告提示。
func Warning(c *gin.Context, msg string) { put(c, LevelWarning, msg) }

// Pop 读取并清除当前 Flash 消息。
func Pop(c *gin.Context) *Message {
	mgr := getSessionManager(c)
	if mgr == nil {
		return nil
	}
	v := mgr.Pop(c.Request.Context(), sessionKey)
	if v == nil {
		return nil
	}
	msg, ok := v.(Message)
	if !ok {
		return nil
	}
	return &msg
}

func put(c *gin.Context, level Level, text string) {
	mgr := getSessionManager(c)
	if mgr == nil {
		logger.FromContext(c.Request.Context()).Warn("flash set skipped: session manager not configured")
		return
	}
	mgr.Put(c.Request.Context(), sessionKey, Message{Level: level, Text: text})
}

func getSessionManager(c *gin.Context) *scs.SessionManager {
	return defaultSessionMgr
}
