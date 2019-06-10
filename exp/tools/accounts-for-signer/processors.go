package main

import (
	"context"
	"sync"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/xdr"
)

type SimpleProcessor struct {
	sync.Mutex
	callCount int
}

func (n *SimpleProcessor) IsConcurrent() bool {
	return false
}

func (n *SimpleProcessor) RequiresInput() bool {
	return true
}

func (n *SimpleProcessor) IncrementAndReturnCallCount() int {
	n.Lock()
	defer n.Unlock()
	n.callCount++
	return n.callCount
}

type AccountsForSignerProcessor struct {
	SimpleProcessor

	Signer string
}

func (p *AccountsForSignerProcessor) ProcessState(ctx context.Context, store *pipeline.Store, r io.StateReadCloser, w io.StateWriteCloser) error {
	defer r.Close()
	defer w.Close()

	for {
		entry, err := r.Read()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}

		if entry.Data.Type != xdr.LedgerEntryTypeAccount {
			continue
		}

		for _, signer := range entry.Data.Account.Signers {
			if signer.Key.Address() == p.Signer {
				err := w.Write(entry)
				if err != nil {
					if err == io.ErrClosedPipe {
						// Reader does not need more data
						return nil
					}
					return err
				}
				break
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	return nil
}

func (p *AccountsForSignerProcessor) Name() string {
	return "AccountsForSignerProcessor"
}

func (n *AccountsForSignerProcessor) IsConcurrent() bool {
	return true
}
