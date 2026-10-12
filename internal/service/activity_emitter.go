package service

import (
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/transport/websocket"
)

type ActivityEmitter struct {
	ch     chan *entity.ActivityLog
	hub    *websocket.Hub
	db     *gorm.DB
	log    *zap.SugaredLogger
	closed bool
	mu     sync.Mutex
}

func NewActivityEmitter(db *gorm.DB, hub *websocket.Hub, log *zap.SugaredLogger, bufferSize int) *ActivityEmitter {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	return &ActivityEmitter{
		ch:  make(chan *entity.ActivityLog, bufferSize),
		hub: hub,
		db:  db,
		log: log,
	}
}

// Emit is non-blocking — business logic does not wait.
func (e *ActivityEmitter) Emit(log *entity.ActivityLog) {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	e.mu.Unlock()

	select {
	case e.ch <- log:
	default:
		e.log.Warnw("activity emitter buffer full, dropping log", "type", log.Type)
	}
}

// StartWorkers processes activity logs asynchronously from the channel.
func (e *ActivityEmitter) StartWorkers(workerCount int) {
	for i := 0; i < workerCount; i++ {
		go e.worker()
	}
}

func (e *ActivityEmitter) worker() {
	for log := range e.ch {
		if err := e.db.Create(log).Error; err != nil {
			e.log.Errorw("failed to persist activity log", "error", err, "type", log.Type)
		}

		if e.hub != nil {
			payload := model.EventPayload[any]{
				Type:    "ACTIVITY",
				Message: log.Title,
				Data:    log,
			}
			e.hub.BroadcastToDapur(log.DapurID, payload)
		}
	}
}

func (e *ActivityEmitter) Shutdown() {
	e.mu.Lock()
	e.closed = true
	e.mu.Unlock()
	close(e.ch)
}
