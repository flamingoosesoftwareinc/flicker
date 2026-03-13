package multipleproviders

import (
	"context"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
)

type Request struct {
	Intervals []time.Duration `json:"intervals"`
}

type Response struct {
	Cycles int `json:"cycles"`
}

type pollingWorkflow struct {
	wc  *flicker.WorkflowContext
	rng *flicker.Provider[int]
}

var _ = flicker.Define[Request, Response]("polling-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &pollingWorkflow{
		wc:  wc,
		rng: flicker.NewProvider[int](wc, "jitter", func() (int, error) { return 42, nil }),
	}
})

func (w *pollingWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	// Create a second provider in Execute body.
	nonce := flicker.NewProvider[string](w.wc, "nonce", func() (string, error) { return "abc", nil })
	_ = nonce

	// First cycle: check time, do work, sleep.
	start, err := w.wc.Time.Now(ctx)
	if err != nil {
		return Response{}, err
	}

	_, err = flicker.Run[string](ctx, w.wc, "poll-1", func(ctx context.Context) (*string, error) {
		s := "ok"
		return &s, nil
	})
	if err != nil {
		return Response{}, err
	}

	err = w.wc.SleepUntil(ctx, start.Add(1*time.Hour))
	if err != nil {
		return Response{}, err
	}

	// Second cycle: check time again, do work, sleep again.
	mid, err := w.wc.Time.Now(ctx)
	if err != nil {
		return Response{}, err
	}

	_, err = flicker.Run[string](ctx, w.wc, "poll-2", func(ctx context.Context) (*string, error) {
		s := "ok"
		return &s, nil
	})
	if err != nil {
		return Response{}, err
	}

	err = w.wc.SleepUntil(ctx, mid.Add(2*time.Hour))
	if err != nil {
		return Response{}, err
	}

	// Third cycle: final check and work.
	_, err = w.wc.Time.Now(ctx)
	if err != nil {
		return Response{}, err
	}

	_, err = flicker.Run[bool](ctx, w.wc, "finalize", func(ctx context.Context) (*bool, error) {
		b := true
		return &b, nil
	})
	if err != nil {
		return Response{}, err
	}

	return Response{Cycles: 3}, nil
}
