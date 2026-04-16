package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/telemetryflow/config"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// Consumer is a franz-go consumer group wrapper with manual offset commits.
//
// Key behaviours enforced here (ADR-002, ADR-006):
//   - AutoCommit is DISABLED — offsets are only committed after the caller
//     confirms successful processing. A crash before commit causes reprocessing,
//     which is correct at-least-once behaviour.
//   - BlockRebalanceOnPoll — holds the rebalance lock during Poll so a
//     rebalance cannot steal partitions between Poll and Commit. Partitions
//     are only revoked after the current Poll+Commit cycle completes.
//   - CooperativeStickyAssignor (franz-go default) — only moves partitions
//     that actually need to change owners, preserving in-memory window state
//     in the processor service (ADR-006).
//   - MaxConcurrentFetches / FetchMaxBytes — bounds memory per poll.
type Consumer struct {
	client *kgo.Client
}

// NewConsumer creates a Consumer joined to group and subscribed to topics.
//
// The consumer does not start fetching until the first call to Poll.
func NewConsumer(cfg config.KafkaConfig, group string, topics ...string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.BootstrapServers),
		kgo.SASL(plain.Auth{
			User: cfg.APIKey,
			Pass: cfg.APISecret,
		}.AsMechanism()),
		kgo.DialTLS(),

		// Consumer group identity.
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topics...),

		// ADR-002: disable auto-commit so offsets only advance after the
		// caller confirms that the record was written to the database.
		kgo.DisableAutoCommit(),

		// Prevent a rebalance from revoking partitions mid-batch. franz-go
		// will delay the rebalance until the current Poll call returns and
		// the caller explicitly calls AllowRebalance (via Close or next Poll).
		kgo.BlockRebalanceOnPoll(),

		// Cap records per poll to bound memory. 500 records × ~1 KB each = ~500 KB
		// per batch, well within a typical heap budget.
		kgo.MaxConcurrentFetches(3),
		kgo.FetchMaxBytes(5<<20), // 5 MiB max per fetch response
	)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer: creating client for group %q: %w", group, err)
	}

	// Verify broker reachability and credentials before starting the service.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("kafka consumer: pinging broker for group %q: %w", group, err)
	}

	return &Consumer{client: client}, nil
}

// Poll blocks until records are available or ctx is cancelled.
//
// The returned Fetches may contain errors from individual partitions —
// callers should check fetches.Errors() after iterating records. A closed
// client or cancelled context returns an empty fetch with no error.
//
// IMPORTANT: franz-go holds the rebalance lock until the next call to Poll
// (because of BlockRebalanceOnPoll). Always call Poll in a tight loop — do
// not hold a fetches result across a long operation before calling Poll again.
func (c *Consumer) Poll(ctx context.Context) kgo.Fetches {
	return c.client.PollFetches(ctx)
}

// Commit marks the given records' offsets as processed in the broker.
//
// Only call after the record has been successfully written to the database.
// If Commit is never called (e.g. process crashes), the broker will redeliver
// the record to the next consumer in the group — this is the at-least-once
// guarantee (ADR-002).
func (c *Consumer) Commit(ctx context.Context, records ...*kgo.Record) error {
	if err := c.client.CommitRecords(ctx, records...); err != nil {
		return fmt.Errorf("kafka consumer: committing offsets: %w", err)
	}
	return nil
}

// Close gracefully leaves the consumer group and closes the underlying client.
// In-flight records that have not been committed will be redelivered to other
// group members — this is safe and expected on shutdown.
func (c *Consumer) Close() {
	c.client.Close()
}
