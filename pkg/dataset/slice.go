package dataset

import (
	"context"

	"inspectgo/pkg/core"
)

type SliceDataset struct {
	NameHint string
	Items    []core.Sample
}

func NewSliceDataset(samples []core.Sample, name string) *SliceDataset {
	if name == "" {
		name = "retry"
	}
	return &SliceDataset{NameHint: name, Items: samples}
}

func (d *SliceDataset) Name() string {
	return d.NameHint
}

func (d *SliceDataset) Len(ctx context.Context) (int, error) {
	return len(d.Items), nil
}

func (d *SliceDataset) Samples(ctx context.Context) (<-chan core.Sample, <-chan error) {
	sampleCh := make(chan core.Sample)
	errCh := make(chan error, 1)
	go func() {
		defer close(sampleCh)
		defer close(errCh)
		for _, s := range d.Items {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case sampleCh <- s:
			}
		}
	}()
	return sampleCh, errCh
}
