package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	error2 "github.com/xBlaz3kx/distributed-scheduler/internal/pkg/error"
	"gopkg.in/guregu/null.v4"
)

func TestJobTypeValid(t *testing.T) {
	var jobType JobType = "INVALID"

	if jobType.Valid() {
		t.Error("Expected false, got true")
	}

	jobType = JobTypeHTTP

	if !jobType.Valid() {
		t.Error("Expected true, got false")
	}
}

func TestJobStatusValid(t *testing.T) {
	var jobStatus JobStatus = "INVALID"

	if jobStatus.Valid() {
		t.Error("Expected false, got true")
	}

	jobStatus = JobStatusRunning

	if !jobStatus.Valid() {
		t.Error("Expected true, got false")
	}
}

func TestAuthTypeValid(t *testing.T) {
	var authType AuthType = "INVALID"

	if authType.Valid() {
		t.Error("Expected false, got true")
	}

	authType = AuthTypeNone

	if !authType.Valid() {
		t.Error("Expected true, got false")
	}
}

func TestHTTPJobValidate(t *testing.T) {
	tests := []struct {
		name string
		job  HTTPJob
		want error
	}{
		{
			name: "valid job",
			job: HTTPJob{
				URL:    "https://example.com",
				Method: "GET",
				Auth: Auth{
					Type: AuthTypeNone,
				},
			},
			want: nil,
		},
		{
			name: "invalid job: empty URL",
			job: HTTPJob{
				URL:    "",
				Method: "GET",
				Auth: Auth{
					Type: AuthTypeNone,
				},
			},
			want: error2.ErrEmptyHTTPJobURL,
		},
		{
			name: "invalid job: empty Method",
			job: HTTPJob{
				URL:    "https://example.com",
				Method: "",
				Auth: Auth{
					Type: AuthTypeNone,
				},
			},
			want: error2.ErrEmptyHTTPJobMethod,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.job.Validate()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAMQPJobValidate(t *testing.T) {
	tests := []struct {
		name string
		job  AMQPJob
		want error
	}{
		{
			name: "valid job",
			job: AMQPJob{
				Connection: "amqp://guest:guest@localhost:5672/",
				Exchange:   "my_exchange",
				RoutingKey: "my_routing_key",
			},
			want: nil,
		},
		{
			name: "invalid job: empty Exchange",
			job: AMQPJob{
				Connection: "amqp://guest:guest@localhost:5672/",
				Exchange:   "",
				RoutingKey: "my_routing_key",
			},
			want: error2.ErrEmptyExchange,
		},
		{
			name: "invalid job: empty RoutingKey",
			job: AMQPJob{
				Connection: "amqp://guest:guest@localhost:5672/",
				Exchange:   "my_exchange",
				RoutingKey: "",
			},
			want: error2.ErrEmptyRoutingKey,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.job.Validate()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAuthValidate(t *testing.T) {
	tests := []struct {
		name string
		auth Auth
		want error
	}{
		{
			name: "valid auth: no auth",
			auth: Auth{
				Type: AuthTypeNone,
			},
			want: nil,
		},
		{
			name: "valid auth: basic auth",
			auth: Auth{
				Type:     AuthTypeBasic,
				Username: null.StringFrom("testuser"),
				Password: null.StringFrom("testpassword"),
			},
			want: nil,
		},
		{
			name: "invalid auth: missing username",
			auth: Auth{
				Type:     AuthTypeBasic,
				Password: null.StringFrom("testpassword"),
			},
			want: error2.ErrEmptyUsername,
		},
		{
			name: "invalid auth: missing password",
			auth: Auth{
				Type:     AuthTypeBasic,
				Username: null.StringFrom("testuser"),
			},
			want: error2.ErrEmptyPassword,
		},
		{
			name: "invalid auth: unsupported auth type",
			auth: Auth{
				Type: "unsupported_type",
			},
			want: error2.ErrInvalidAuthType,
		},
		{
			name: "valid auth: bearer token",
			auth: Auth{
				Type:        AuthTypeBearer,
				BearerToken: null.StringFrom("testtoken"),
			},
			want: nil,
		},
		{
			name: "invalid auth: missing bearer token",
			auth: Auth{
				Type: AuthTypeBearer,
			},
			want: error2.ErrEmptyBearerToken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.auth.Validate()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestJobValidate(t *testing.T) {
	tests := []struct {
		name string
		job  Job
		want error
	}{
		{
			name: "valid job",
			job: Job{
				ID:        uuid.New(),
				Type:      JobTypeHTTP,
				Status:    JobStatusRunning,
				ExecuteAt: null.TimeFrom(time.Now().Add(time.Minute)),
				HTTPJob: &HTTPJob{
					URL:    "https://example.com",
					Method: "GET",
					Auth: Auth{
						Type: AuthTypeNone,
					},
				},
				CreatedAt: time.Now(),
			},
			want: nil,
		},
		{
			name: "invalid job: missing ID",
			job: Job{
				Type:      JobTypeHTTP,
				Status:    JobStatusRunning,
				ExecuteAt: null.TimeFrom(time.Now().Add(time.Minute)),
				HTTPJob: &HTTPJob{
					URL:    "https://example.com",
					Method: "GET",
					Auth: Auth{
						Type: AuthTypeNone,
					},
				},
				CreatedAt: time.Now(),
			},
			want: error2.ErrInvalidJobID,
		},
		{
			name: "invalid job: http type with nil HTTPJob",
			job: Job{
				ID:        uuid.New(),
				Type:      JobTypeHTTP,
				Status:    JobStatusRunning,
				ExecuteAt: null.TimeFrom(time.Now().Add(time.Minute)),
				CreatedAt: time.Now(),
			},
			want: error2.ErrHTTPJobNotDefined,
		},
		{
			name: "invalid job: unsupported Type",
			job: Job{
				ID:        uuid.New(),
				Type:      "invalid_type",
				Status:    JobStatusRunning,
				ExecuteAt: null.TimeFrom(time.Now().Add(time.Minute)),
				HTTPJob: &HTTPJob{
					URL:    "https://example.com",
					Method: "GET",
					Auth: Auth{
						Type: AuthTypeNone,
					},
				},
				CreatedAt: time.Now(),
			},
			want: error2.ErrInvalidJobType,
		},
		{
			name: "invalid job: invalid cron expression",
			job: Job{
				ID:           uuid.New(),
				Type:         JobTypeHTTP,
				Status:       JobStatusRunning,
				CronSchedule: null.StringFrom("invalid_cron_expression"),
				HTTPJob: &HTTPJob{
					URL:    "https://example.com",
					Method: "GET",
					Auth: Auth{
						Type: AuthTypeNone,
					},
				},
				CreatedAt: time.Now(),
			},
			want: error2.ErrInvalidCronSchedule,
		},
		{
			name: "invalid job: schedule and execute at both defined",
			job: Job{
				ID:           uuid.New(),
				Type:         JobTypeHTTP,
				Status:       JobStatusRunning,
				CronSchedule: null.StringFrom("* * * * *"),
				ExecuteAt:    null.TimeFrom(time.Now().Add(time.Minute)),
				HTTPJob: &HTTPJob{
					URL:    "https://example.com",
					Method: "GET",
					Auth: Auth{
						Type: AuthTypeNone,
					},
				},
				CreatedAt: time.Now(),
			},
			want: error2.ErrInvalidJobSchedule,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.job.Validate()
			assert.Equal(t, tc.want, got)
		})
	}
}
