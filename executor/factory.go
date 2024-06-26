package executor

import (
	"fmt"
	"github.com/xBlaz3kx/distributed-scheduler/model"
)

type Factory interface {
	NewExecutor(job *model.Job, options ...Option) (model.Executor, error)
}

type factory struct {
	client HttpClient
}

func NewFactory(client HttpClient) Factory {
	return &factory{
		client: client,
	}
}

// Option is a function that modifies an executor before it is returned (e.g. WithRetry)
type Option func(executor model.Executor) model.Executor

func (f *factory) NewExecutor(job *model.Job, options ...Option) (model.Executor, error) {

	var executor model.Executor
	switch job.Type {
	case model.JobTypeHTTP:
		executor = &hTTPExecutor{Client: f.client}
	case model.JobTypeAMQP:
		executor = &aMQPExecutor{}
	default:
		return nil, fmt.Errorf("unknown job type: %v", job.Type)
	}

	for _, option := range options {
		executor = option(executor)
	}

	return executor, nil
}
