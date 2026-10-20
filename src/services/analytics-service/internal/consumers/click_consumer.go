package consumers

import (
	"analytics-service/internal/metrics"
	"analytics-service/internal/models"
	"analytics-service/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	reconnectDelay = 5 * time.Second
	flushTimeout   = 10 * time.Second
)

type Config struct {
	URL           string
	Exchange      string
	Queue         string
	RoutingKey    string
	Prefetch      int
	BatchSize     int
	FlushInterval time.Duration
}

// ClickService persists a batch of consumed events.
type ClickService interface {
	InsertClickEventBatch(ctx context.Context, events []models.ClickEvent) error
}

type ClickConsumer struct {
	cfg     Config
	service ClickService
	done    chan struct{}
	wg      sync.WaitGroup
}

func New(cfg Config, service ClickService) *ClickConsumer {
	return &ClickConsumer{
		cfg:     cfg,
		service: service,
		done:    make(chan struct{}),
	}
}

func (c *ClickConsumer) Start() {
	c.wg.Add(1)
	go c.run()
}

func (c *ClickConsumer) Close() {
	close(c.done)
	c.wg.Wait()

	logger.Log.Info("click consumer stopped")
}

// run keeps a consume session alive, reconnecting until Close is called.
func (c *ClickConsumer) run() {
	defer c.wg.Done()

	for {
		if err := c.consume(); err != nil {
			logger.Log.Warn("consumer session ended, reconnecting", zap.Error(err))
		}

		select {
		case <-c.done:
			return
		case <-time.After(reconnectDelay):
		}
	}
}

// consume runs one session: connect, declare, consume until the connection
// drops or Close is called.
func (c *ClickConsumer) consume() error {
	connection, err := amqp.Dial(c.cfg.URL)
	if err != nil {
		return err
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	if err := c.declare(channel); err != nil {
		return err
	}

	deliveries, err := channel.Consume(c.cfg.Queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	logger.Log.Info("consuming click events",
		zap.String("queue", c.cfg.Queue),
		zap.Int("prefetch", c.cfg.Prefetch),
		zap.Int("batch_size", c.cfg.BatchSize),
	)

	return c.loop(deliveries)
}

// declare is idempotent, so every replica can safely run it at startup.
func (c *ClickConsumer) declare(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(c.cfg.Exchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}

	if _, err := channel.QueueDeclare(c.cfg.Queue, true, false, false, false, nil); err != nil {
		return err
	}

	if err := channel.QueueBind(c.cfg.Queue, c.cfg.RoutingKey, c.cfg.Exchange, false, nil); err != nil {
		return err
	}

	return channel.Qos(c.cfg.Prefetch, 0, false)
}

func (c *ClickConsumer) loop(deliveries <-chan amqp.Delivery) error {
	events := make([]models.ClickEvent, 0, c.cfg.BatchSize)
	var last amqp.Delivery

	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			c.flush(events, last)
			return nil

		case delivery, ok := <-deliveries:
			if !ok {
				c.flush(events, last)
				return errors.New("rabbitmq closed the delivery channel")
			}

			var event models.ClickEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				metrics.ClicksConsumedTotal.WithLabelValues("malformed").Inc()
				logger.Log.Warn("discarding malformed click event", zap.Error(err))
				_ = delivery.Ack(false)
				continue
			}

			metrics.ClicksConsumedTotal.WithLabelValues("ok").Inc()
			events = append(events, event)
			last = delivery

			if len(events) >= c.cfg.BatchSize {
				c.flush(events, last)
				events = events[:0]
			}

		case <-ticker.C:
			if len(events) > 0 {
				c.flush(events, last)
				events = events[:0]
			}
		}
	}
}

// flush persists the batch, then acknowledges every message up to the last one.
func (c *ClickConsumer) flush(events []models.ClickEvent, last amqp.Delivery) {
	if len(events) == 0 {
		return
	}

	metrics.ClicksBatchSize.Observe(float64(len(events)))

	ctx, cancel := context.WithTimeout(context.Background(), flushTimeout)
	defer cancel()

	start := time.Now()
	err := c.service.InsertClickEventBatch(ctx, events)
	metrics.ClicksFlushDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.ClicksFlushTotal.WithLabelValues("requeued").Inc()
		logger.Log.Error("failed to persist clicks, requeueing", zap.Error(err))
		_ = last.Nack(true, true)
		return
	}

	metrics.ClicksFlushTotal.WithLabelValues("ok").Inc()
	if err := last.Ack(true); err != nil {
		logger.Log.Warn("failed to acknowledge clicks", zap.Error(err))
	}
}
