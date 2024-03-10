package executor

import (
	"github.com/stretchr/testify/assert"
	"github.com/xBlaz3kx/distributed-scheduler/model"
	"net/http"
	"testing"
)

func TestNewExecutor(t *testing.T) {
	j := &model.Job{
		Type: model.JobTypeHTTP,
		HTTPJob: &model.HTTPJob{
			Method: "GET",
			URL:    "http://www.example.com",
		},
	}

	factory := NewFactory(&http.Client{})

	executor, err := factory.NewExecutor(j)
	assert.Nil(t, err)
	assert.IsType(t, &hTTPExecutor{}, executor)

	j.Type = model.JobTypeAMQP
	executor, err = factory.NewExecutor(j)
	assert.Nil(t, err)
	assert.IsType(t, &aMQPExecutor{}, executor)

	j.Type = "unknown"
	executor, err = factory.NewExecutor(j)
	assert.NotNil(t, err)
	assert.Nil(t, executor)

	j.Type = model.JobTypeHTTP
	executor, err = factory.NewExecutor(j, WithRetry)
	assert.Nil(t, err)
	assert.IsType(t, &retryExecutor{}, executor)
}
