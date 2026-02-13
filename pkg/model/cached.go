package model

import (
	"context"

	"inspectgo/pkg/cache"
	"inspectgo/pkg/core"
)

type CachedModel struct {
	Model core.Model
	Cache *cache.Cache
}

func (c CachedModel) Name() string {
	if c.Model == nil {
		return ""
	}
	return c.Model.Name()
}

func (c CachedModel) Generate(ctx context.Context, prompt string, opts core.GenerateOptions) (core.Response, error) {
	if c.Model == nil {
		return core.Response{}, nil
	}
	if c.Cache != nil {
		if resp, ok := c.Cache.Get(c.Name(), prompt, opts); ok {
			return resp, nil
		}
	}
	resp, err := c.Model.Generate(ctx, prompt, opts)
	if err != nil {
		return core.Response{}, err
	}
	if c.Cache != nil {
		_ = c.Cache.Set(c.Name(), prompt, opts, resp)
	}
	return resp, nil
}
