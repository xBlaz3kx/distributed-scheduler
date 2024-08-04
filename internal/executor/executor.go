package executor

import (
	"context"

	"github.com/xBlaz3kx/distributed-scheduler/internal/model"
)

type Executor interface {
	Execute(ctx context.Context, job *model.Job) error
}
