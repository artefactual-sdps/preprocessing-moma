package workercmd

import (
	"context"

	"github.com/artefactual-sdps/temporal-activities/removefiles"
	"github.com/go-logr/logr"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_interceptor "go.temporal.io/sdk/interceptor"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"

	"github.com/artefactual-sdps/preprocessing-moma/internal/config"
	"github.com/artefactual-sdps/preprocessing-moma/internal/workflow"
)

const Name = "preprocessing-worker"

type Main struct {
	logger         logr.Logger
	cfg            config.Configuration
	temporalWorker temporalsdk_worker.Worker
	temporalClient temporalsdk_client.Client
}

func NewMain(logger logr.Logger, cfg config.Configuration) *Main {
	return &Main{
		logger: logger,
		cfg:    cfg,
	}
}

func (m *Main) Run(ctx context.Context) error {
	c, err := temporalsdk_client.Dial(temporalsdk_client.Options{
		HostPort:  m.cfg.Temporal.Address,
		Namespace: m.cfg.Temporal.Namespace,
		Logger:    temporal.Logger(m.logger.WithName("temporal")),
	})
	if err != nil {
		m.logger.Error(err, "Unable to create Temporal client.")
		return err
	}
	m.temporalClient = c

	w := temporalsdk_worker.New(m.temporalClient, m.cfg.Temporal.TaskQueue, temporalsdk_worker.Options{
		EnableSessionWorker:               true,
		MaxConcurrentSessionExecutionSize: m.cfg.Worker.MaxConcurrentSessions,
		Interceptors: []temporalsdk_interceptor.WorkerInterceptor{
			temporal.NewLoggerInterceptor(m.logger.WithName("worker")),
		},
	})
	m.temporalWorker = w

	w.RegisterWorkflowWithOptions(
		workflow.NewPreprocessingWorkflow(m.cfg.SharedPath).Execute,
		temporalsdk_workflow.RegisterOptions{Name: m.cfg.Temporal.WorkflowName},
	)
	w.RegisterActivityWithOptions(
		removefiles.NewActivity(m.cfg.RemoveFiles).Execute,
		temporalsdk_activity.RegisterOptions{Name: removefiles.ActivityName},
	)

	if err := w.Start(); err != nil {
		m.logger.Error(err, "Worker failed to start or fatal error during its execution.")
		return err
	}

	return nil
}

func (m *Main) Close() error {
	if m.temporalWorker != nil {
		m.temporalWorker.Stop()
	}

	if m.temporalClient != nil {
		m.temporalClient.Close()
	}

	return nil
}
