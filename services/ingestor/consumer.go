package main

// ingestorConsumer wraps pkg/kafka.Consumer and coordinates the poll → batch → flush loop.
//
// The loop:
//  1. Poll a batch of records from device.metrics (up to MaxPollRecords=500)
//  2. Deserialize each record from Avro wire format into DeviceMetric
//  3. Accumulate into the current batch
//  4. When batch is full OR flush ticker fires: hand batch to the writer
//  5. On successful write: commit offsets for all records in the batch
//  6. On failure: send to DLQ, do NOT commit offsets
//
// The ticker-based flush (step 4) is implemented with a select over a
// time.Ticker channel — never time.Sleep (see CLAUDE.md "What NOT To Do").
type ingestorConsumer struct {
	// TODO: *kafka.Consumer, *writer, *kafka.DLQPublisher, *schema.Deserializer
}

// newIngestorConsumer creates and wires up the consumer with its dependencies.
func newIngestorConsumer() (*ingestorConsumer, error) {
	panic("not implemented — implement in Phase 2")
}

// Run starts the consume-batch-flush loop. Blocks until ctx is cancelled.
func (c *ingestorConsumer) Run() error {
	panic("not implemented — implement in Phase 2")
}

// Shutdown drains the current batch and commits offsets before returning.
func (c *ingestorConsumer) Shutdown() error {
	panic("not implemented — implement in Phase 2")
}
