package service

import (
	"apiChecker/internal/client"
	"apiChecker/internal/dto"

	"context"
	"log"
	"time"
)

type Batcher struct {
	catalogClient *client.CatalogClient
	inChan        <-chan dto.CheckItem
	batchSize     int
	flushInterval time.Duration
	buffer        []dto.CheckItem
}

func NewBatcher(
	catalogClient *client.CatalogClient,
	inChan <-chan dto.CheckItem,
	batchSize int,
	flushInterval time.Duration,
) *Batcher {
	if batchSize <= 0 {
		batchSize = 50
	}
	if flushInterval <= 0 {
		flushInterval = 2 * time.Second
	}

	return &Batcher{
		catalogClient: catalogClient,
		inChan:        inChan,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		buffer:        make([]dto.CheckItem, 0, batchSize),
	}
}

func (b *Batcher) Run(ctx context.Context) {
	ticker := time.NewTicker(b.flushInterval)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.flushWithTimeout()
			return

		case <-ticker.C:
			if len(b.buffer) > 0 {
				b.flush(ctx)
			}

		case item, ok := <-b.inChan:
			if !ok {
				b.flushWithTimeout()
				return
			}

			b.buffer = append(b.buffer, item)

			if len(b.buffer) >= b.batchSize {
				b.flush(ctx)
				ticker.Reset(b.flushInterval)
			}
		}
	}
}

func (b *Batcher) flush(ctx context.Context) {
	if len(b.buffer) == 0 {
		return
	}

	req := &dto.SaveBatchRequest{
		Results: b.buffer,
	}

	if err := b.catalogClient.SendBatchChecks(ctx, req); err != nil {
		log.Printf("failed to send batch of %d checks to catalog: %v", len(b.buffer), err)
	}

	b.buffer = b.buffer[:0]
}

func (b *Batcher) flushWithTimeout() {
	if len(b.buffer) == 0 {
		return
	}
	flushCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	b.flush(flushCtx)
}
