package kafka

// DLQPublisher routes unprocessable messages to a dead-letter topic (ADR-007).
//
// A message goes to the DLQ after 3 failed processing attempts. The DLQ record
// preserves the original payload and attaches error metadata as Kafka headers:
//
//	error_reason      — human-readable description of why processing failed
//	failed_at         — RFC3339 timestamp of the final failure
//	original_topic    — topic the message was originally consumed from
//	original_partition — partition number (as a string)
//	original_offset   — offset of the failed record (as a string)
//
// This allows the failed message to be inspected and replayed without
// redeploying the service. The DLQ topic is separate from the main topic
// so a poison pill cannot block the main pipeline's partition.
//
// If publishing to the DLQ itself fails, the error is logged and an error
// counter is incremented. The DLQ failure must NEVER block the main pipeline
// (see CLAUDE.md "What NOT To Do").
//
// Usage:
//
//	dlq, err := NewDLQPublisher(cfg.Kafka, TopicMetricsDLQ)
//	defer dlq.Close()
//
//	if err := process(record); err != nil {
//	    if dlqErr := dlq.Publish(ctx, record, err); dlqErr != nil {
//	        slog.ErrorContext(ctx, "dlq publish failed", "error", dlqErr)
//	        // do not return — continue processing other records
//	    }
//	}
type DLQPublisher struct {
	// TODO: embed *Producer
}

// NewDLQPublisher creates a DLQPublisher that sends failed records to dlqTopic.
func NewDLQPublisher() (*DLQPublisher, error) {
	panic("not implemented — implement in Phase 1")
}

// Publish sends the original record to the DLQ topic with error metadata headers.
// This is fire-and-forget from the caller's perspective — errors are logged internally.
func (d *DLQPublisher) Publish() error {
	panic("not implemented — implement in Phase 1")
}

// Close flushes and closes the underlying producer.
func (d *DLQPublisher) Close() {
	panic("not implemented — implement in Phase 1")
}
