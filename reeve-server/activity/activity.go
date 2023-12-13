package activity

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/reeveci/reeve-lib/schema"
)

func NewRuntimeActivity(workerGroup string, timeout time.Duration, notifications chan<- schema.PipelineStatus) *RuntimeActivity {
	return &RuntimeActivity{
		workerGroup:   workerGroup,
		Timeout:       timeout,
		status:        make(map[string]*RuntimeStatus),
		notifications: notifications,
	}
}

type RuntimeActivity struct {
	lock sync.Mutex

	Timeout time.Duration

	workerGroup   string
	status        map[string]*RuntimeStatus
	notifications chan<- schema.PipelineStatus
}

type PipelineActivity struct {
	schema.Pipeline
	ActivityID string
}

func (r *RuntimeActivity) RegisterPipeline(pipeline schema.Pipeline) (activity PipelineActivity) {
	activity.Pipeline = pipeline
	activity.ActivityID = uuid.NewString()

	defer r.NotifyUpdate(activity.ActivityID)

	r.lock.Lock()
	defer r.lock.Unlock()

	status := RuntimeStatus{
		PipelineStatus: schema.PipelineStatus{
			Pipeline:    censorSecrets(pipeline),
			WorkerGroup: r.workerGroup,
			ActivityID:  activity.ActivityID,

			Status: schema.STATUS_ENQUEUED,
		},

		notifyTimeout: func() {
			r.NotifyUpdate(activity.ActivityID)
		},
	}
	status.Result.ExitCode = -1

	r.status[activity.ActivityID] = &status
	return
}

func censorSecrets(pipeline schema.Pipeline) schema.Pipeline {
	censoredEnv := make(map[string]schema.Env, len(pipeline.Env))

	for key, env := range pipeline.Env {
		if env.Secret {
			env.Value = "*******"
		}

		censoredEnv[key] = env
	}

	pipeline.Env = censoredEnv
	return pipeline
}

func (r *RuntimeActivity) Status(id string) *RuntimeStatus {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.status[id]
}

func (r *RuntimeActivity) NotifyUpdate(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	status := r.status[id]
	if status == nil {
		return
	}

	status.Lock()
	if status.Finished() {
		if status.Logs != nil {
			status.Logs.Close()
		}

		delete(r.status, id)
	}
	notification := status.PipelineStatus
	status.Unlock()

	r.notifications <- notification
}

type RuntimeStatus struct {
	schema.PipelineStatus

	notifyTimeout func()

	sync.Mutex
	cancel context.CancelFunc
}

func (r *RuntimeStatus) ResetTimeout(timeout time.Duration) {
	r.ClearTimeout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	r.cancel = cancel

	go func() {
		<-ctx.Done()

		if ctx.Err() == context.DeadlineExceeded {
			r.Lock()

			if r.Running() {
				r.Status = schema.STATUS_TIMEOUT
				r.Unlock()

				r.notifyTimeout()
			} else {
				r.Unlock()
			}
		}
	}()
}

func (r *RuntimeStatus) ClearTimeout() {
	if r.cancel != nil {
		r.cancel()
	}
}
