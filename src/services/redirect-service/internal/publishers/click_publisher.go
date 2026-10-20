package publishers

import (
	"context"
	"encoding/json"
	"errors"
	"redirect-service/internal/models"
	"redirect-service/pkg/logger"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	publishTimeout = 2 * time.Second
	reconnectDelay = 5 * time.Second
)

var errBrokerDown = errors.New("rabbitmq unavailable")

type Config struct {
	URL        string
	Exchange   string
	RoutingKey string
	BufferSize int
}

// for click mode off
type Noop struct{}

func (Noop) Publish(models.ClickEvent) {}

type ClickPublisher struct {
	cfg       Config
	events    chan models.ClickEvent
	done      chan struct{}
	wg        sync.WaitGroup
	dropped   atomic.Int64
	published atomic.Int64

	connection *amqp.Connection
	channel    *amqp.Channel
	nextRetry  time.Time
}

func New(cfg Config) *ClickPublisher {
	return &ClickPublisher{
		cfg:    cfg,
		events: make(chan models.ClickEvent, cfg.BufferSize),
		done:   make(chan struct{}),
	}
}

// Publish enqueues an event. It never blocks and never fails: if the buffer is
// full the event is dropped and counted, because a slow broker must not slow
// down redirects.
func (p *ClickPublisher) Publish(event models.ClickEvent) {
	select {
	case p.events <- event:
	default:
		p.dropped.Add(1)
	}
}

func (p *ClickPublisher) Start() {
	p.wg.Add(1)
	go p.run()
}

func (p *ClickPublisher) Close() {
	close(p.done)
	p.wg.Wait()

	logger.Log.Info("click publisher stopped", zap.Int64("dropped", p.dropped.Load()))
}

// Dropped reports lost events; exported as a Prometheus counter in Phase 5.
func (p *ClickPublisher) Dropped() int64 {
	return p.dropped.Load()
}

func (p *ClickPublisher) run() {
	defer p.wg.Done()
	defer p.disconnect()

	for {
		select {
		case <-p.done:
			return

		case event := <-p.events:
			if err := p.send(event); err != nil {
				p.dropped.Add(1)
			}
		}
	}
}

// send connects if needed, then publishes one event.
func (p *ClickPublisher) send(event models.ClickEvent) error {
	if p.channel == nil {
		if time.Now().Before(p.nextRetry) {
			return errBrokerDown
		}
		if err := p.connect(); err != nil {
			p.nextRetry = time.Now().Add(reconnectDelay)
			logger.Log.Warn("rabbitmq unavailable, dropping click events", zap.Error(err))
			return err
		}
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()

	err = p.channel.PublishWithContext(ctx,
		p.cfg.Exchange, p.cfg.RoutingKey, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Timestamp:   event.OccurredAt,
			Body:        body,
		},
	)
	if err != nil {
		p.disconnect()
		p.nextRetry = time.Now().Add(reconnectDelay)
		logger.Log.Warn("publish failed, will reconnect", zap.Error(err))
		return err
	}

	p.published.Add(1)
	return nil
}

func (p *ClickPublisher) connect() error {
	// create tcp connection
	connection, err := amqp.Dial(p.cfg.URL)
	if err != nil {
		return err
	}

	// create ampq channel on top of tcp
	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return err
	}

	if err := channel.ExchangeDeclare(p.cfg.Exchange, "direct", true, false, false, false, nil); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return err
	}

	p.connection, p.channel = connection, channel
	logger.Log.Info("connected to rabbitmq", zap.String("exchange", p.cfg.Exchange))
	return nil
}

func (p *ClickPublisher) disconnect() {
	if p.channel != nil {
		_ = p.channel.Close()
		p.channel = nil
	}
	if p.connection != nil {
		_ = p.connection.Close()
		p.connection = nil
	}
}

// metrics for prometheus
func (p *ClickPublisher) Published() int64    { return p.published.Load() }
func (p *ClickPublisher) BufferUsed() int     { return len(p.events) }
func (p *ClickPublisher) BufferCapacity() int { return cap(p.events) }
