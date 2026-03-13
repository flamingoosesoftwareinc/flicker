package sleepuntil

import (
	"context"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
)

type Request struct {
	Delay time.Duration `json:"delay"`
}

type Response struct {
	Done bool `json:"done"`
}

type sleepWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[Request, Response]("sleep-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &sleepWorkflow{wc: wc}
})

func (w *sleepWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	now, err := w.wc.Time.Now(ctx)
	if err != nil {
		return Response{}, err
	}

	err = w.wc.SleepUntil(ctx, now.Add(req.Delay))
	if err != nil {
		return Response{}, err
	}

	_, err = flicker.Run[bool](ctx, w.wc, "after-sleep", func(ctx context.Context) (*bool, error) {
		b := true
		return &b, nil
	})
	if err != nil {
		return Response{}, err
	}

	return Response{Done: true}, nil
}
