package iso8583

import (
	"context"
	"fmt"
	"sync"
)

// Processor provides high-level concurrent processing for ISO8583 messages.
// It unpacks raw byte slices into Message structs using a pool of goroutines.
type Processor struct {
	packager     *CompiledPackager // The message specification
	concurrency  int               // Max number of goroutines for processing
	batchSize    int               // (Not currently used)
	errorHandler func(error)       // Callback for handling errors
}

// ProcessorOption defines a function signature for configuring a Processor.
type ProcessorOption func(*Processor)

// WithConcurrency sets the maximum number of concurrent goroutines for the processor.
func WithConcurrency(n int) ProcessorOption {
	return func(p *Processor) {
		p.concurrency = n
	}
}

// WithBatchSize sets the batch size (not currently used in this implementation).
func WithBatchSize(size int) ProcessorOption {
	return func(p *Processor) {
		p.batchSize = size
	}
}

// WithErrorHandler sets a custom error handler for errors encountered during
// batch or stream processing.
func WithErrorHandler(handler func(error)) ProcessorOption {
	return func(p *Processor) {
		p.errorHandler = handler
	}
}

// NewProcessor creates a new Processor with the given packager and options.
func NewProcessor(packager *CompiledPackager, opts ...ProcessorOption) *Processor {
	p := &Processor{
		packager:    packager,
		concurrency: 4,   // Default concurrency
		batchSize:   100, // Default batch size
		errorHandler: func(err error) { // Default error handler
			fmt.Printf("processor error: %v\n", err)
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Process unpacks a single raw ISO8583 message.
func (p *Processor) Process(data []byte) (*Message, error) {
	// Get a new message from the pool (via NewMessage)
	msg := NewMessage(WithPackager(p.packager))

	if err := msg.Unpack(data); err != nil {
		msg.Release() // Release message back to pool on error
		return nil, err
	}

	// Note: The caller is responsible for calling msg.Release() when done.
	return msg, nil
}

// ProcessBatch unpacks a slice of raw messages concurrently.
// It uses a semaphore to limit concurrency to p.concurrency.
func (p *Processor) ProcessBatch(ctx context.Context, dataSlice [][]byte) ([]*Message, error) {
	results := make([]*Message, len(dataSlice))
	errors := make([]error, len(dataSlice))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.concurrency) // Limit concurrent goroutines

	for i, data := range dataSlice {
		// Check for context cancellation before starting a new job
		select {
		case <-ctx.Done():
			// Don't start new jobs if context is cancelled
			wg.Wait() // Wait for already-running jobs
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore slot

		go func(idx int, msgData []byte) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore slot

			// Get message from pool
			msg := NewMessage(WithPackager(p.packager))
			if err := msg.Unpack(msgData); err != nil {
				errors[idx] = err
				if p.errorHandler != nil {
					p.errorHandler(err)
				}
				msg.Release() // Release on error
				return
			}

			results[idx] = msg
		}(i, data)
	}

	wg.Wait() // Wait for all goroutines to finish

	// Check for the first error encountered
	for _, err := range errors {
		if err != nil {
			// Note: This returns partial results, but also an error.
			// The caller must handle releasing messages in the results slice.
			return results, err
		}
	}

	return results, nil
}

// ProcessStream concurrently unpacks messages from an input channel and
// sends the parsed *Message structs to an output channel.
func (p *Processor) ProcessStream(ctx context.Context, input <-chan []byte, output chan<- *Message) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, p.concurrency) // Limit concurrency

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, wait for running jobs and exit
			wg.Wait()
			return ctx.Err()

		case data, ok := <-input:
			if !ok {
				// Input channel closed, wait for running jobs and exit
				wg.Wait()
				return nil
			}

			wg.Add(1)
			semaphore <- struct{}{} // Acquire semaphore

			go func(msgData []byte) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release semaphore

				msg := NewMessage(WithPackager(p.packager))
				if err := msg.Unpack(msgData); err != nil {
					if p.errorHandler != nil {
						p.errorHandler(err)
					}
					msg.Release() // Release on error
					return
				}

				// Send the parsed message to the output channel,
				// or stop if the context is cancelled.
				select {
				case output <- msg:
				case <-ctx.Done():
					msg.Release() // Release if we can't send
				}
			}(data)
		}
	}
}

// Shutdown performs a graceful shutdown (currently a placeholder).
func (p *Processor) Shutdown(ctx context.Context) error {
	// This could be used to close channels, wait for goroutines, etc.
	return nil
}
