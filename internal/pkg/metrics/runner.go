package metrics

import (
	"context"

	"github.com/xBlaz3kx/DevX/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Add attributes: Job Type/Executor, Instance ID, status, numberOfTries

type RunnerMetrics struct {
	enabled bool

	jobsTotal metric.Int64Counter

	jobsExecuted metric.Int64Counter

	jobsFailed metric.Int64Counter

	jobRetries metric.Int64Counter

	jobDuration metric.Float64Histogram

	jobsInExecution metric.Int64Gauge
}

func NewRunnerMetrics(config observability.MetricsConfig) *RunnerMetrics {
	if !config.Enabled {
		return &RunnerMetrics{enabled: false}
	}

	meter := otel.GetMeterProvider().Meter("runner")

	jobsTotal, err := meter.Int64Counter("scheduler_runner_jobs_total")
	must(err)

	jobsExecuted, err := meter.Int64Counter("scheduler_runner_jobs_executed")
	must(err)

	jobsFailed, err := meter.Int64Counter("scheduler_runner_jobs_failed")
	must(err)

	jobRetries, err := meter.Int64Counter("scheduler_runner_job_retries")
	must(err)

	jobDuration, err := meter.Float64Histogram("scheduler_runner_job_duration")
	must(err)

	jobsInExecution, err := meter.Int64Gauge("scheduler_runner_jobs_in_execution")
	must(err)

	return &RunnerMetrics{
		enabled:         true,
		jobsTotal:       jobsTotal,
		jobsExecuted:    jobsExecuted,
		jobsFailed:      jobsFailed,
		jobRetries:      jobRetries,
		jobDuration:     jobDuration,
		jobsInExecution: jobsInExecution,
	}
}

func (r *RunnerMetrics) IncreaseJobsInExecution(ctx context.Context, numJobs int, attributes ...attribute.KeyValue) {
	if r.enabled {
		// Increase gauge metric for number of running jobs
		attrs := metric.WithAttributes(attributes...)
		r.jobsInExecution.Record(ctx, int64(numJobs), attrs)
	}
}

func (r *RunnerMetrics) DecreaseJobsInExecution(ctx context.Context, numJobs int, attributes ...attribute.KeyValue) {
	if r.enabled {
		jobs := int64(numJobs)
		// Increase gauge metric for number of running jobs
		attrs := metric.WithAttributes(attributes...)
		r.jobsInExecution.Record(ctx, -jobs, attrs)
		r.jobsTotal.Add(ctx, jobs, attrs)
		r.jobsExecuted.Add(ctx, jobs, attrs)
	}
}

func (r *RunnerMetrics) RecordJobDuration(ctx context.Context, duration float64, attributes ...attribute.KeyValue) {
	if r.enabled {
		attrs := metric.WithAttributes(attributes...)
		r.jobDuration.Record(ctx, duration, attrs)
	}
}

func (r *RunnerMetrics) IncrementJobRetries(ctx context.Context, attributes ...attribute.KeyValue) {
	if r.enabled {
		attrs := metric.WithAttributes(attributes...)
		r.jobRetries.Add(ctx, 1, attrs)
	}
}

func (r *RunnerMetrics) IncreaseFailedJobCount(ctx context.Context, attributes ...attribute.KeyValue) {
	if r.enabled {
		attrs := metric.WithAttributes(attributes...)
		r.jobsFailed.Add(ctx, 1, attrs)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
