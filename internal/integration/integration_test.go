package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/artefactual-sdps/temporal-activities/removefiles"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	cp "github.com/otiai10/copy"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	"gotest.tools/v3/assert"
	tfs "gotest.tools/v3/fs"

	"github.com/artefactual-sdps/preprocessing-moma/cmd/worker/workercmd"
	"github.com/artefactual-sdps/preprocessing-moma/internal/config"
	"github.com/artefactual-sdps/preprocessing-moma/internal/workflow"
)

type temporalInstance struct {
	client temporalsdk_client.Client
	addr   string // Used when we're connected to a user-provided instance.
}

func setUpTemporal(ctx context.Context, t *testing.T) *temporalInstance {
	t.Helper()

	// Fallback to development server provided by the Temporal GO SDK.
	s, err := temporalsdk_testsuite.StartDevServer(ctx, temporalsdk_testsuite.DevServerOptions{
		LogLevel: "fatal",
	})

	assert.NilError(t, err, "Failed to start Temporal development server.")
	t.Cleanup(func() {
		s.Stop()
	})

	c := s.Client()
	t.Cleanup(func() {
		c.Close()
	})

	return &temporalInstance{
		client: c,
		addr:   s.FrontendHostPort(),
	}
}

func defaultConfig() config.Configuration {
	return config.Configuration{
		Verbosity: 2,
		Debug:     true,
		Worker: config.WorkerConfig{
			MaxConcurrentSessions: 1,
		},
		Temporal: config.Temporal{
			Namespace:    "default",
			TaskQueue:    "preprocessing",
			WorkflowName: "preprocessing",
		},
		RemoveFiles: removefiles.Config{
			RemoveNames: ".DS_Store",
		},
	}
}

func defaultLogger(t *testing.T) logr.Logger {
	t.Helper()

	return testr.NewWithOptions(t, testr.Options{
		LogTimestamp: false,
		Verbosity:    2,
	})
}

type testEnv struct {
	t   *testing.T
	cfg config.Configuration

	testDir *tfs.Dir
}

func newTestEnv(t *testing.T, cfg config.Configuration) *testEnv {
	t.Helper()

	env := &testEnv{t: t, cfg: cfg}
	env.createTestDir()

	return env
}

func (env *testEnv) createTestDir() {
	env.t.Helper()

	env.testDir = tfs.NewDir(env.t, "preprocessing-moma-test")
	env.cfg.SharedPath = env.testDir.Path()
}

func (env *testEnv) startWorker(ctx context.Context) {
	env.t.Helper()

	ctx, cancel := context.WithCancel(ctx)
	m := workercmd.NewMain(defaultLogger(env.t), env.cfg)

	env.t.Cleanup(func() {
		cancel()
		if err := m.Close(); err != nil {
			env.t.Fatal(err)
		}
	})

	done := make(chan error)
	go func() {
		done <- m.Run(ctx)
	}()

	err, ok := <-done
	if ok && err != nil {
		env.t.Fatal(err)
	}
}

func (env *testEnv) copyTestTransfer(name string) {
	env.t.Helper()

	src := filepath.Join("testdata", name)
	dest := env.testDir.Join(name)

	if err := cp.Copy(src, dest); err != nil {
		env.t.Fatalf("Error copying %s to %s", src, dest)
	}
}

func TestIntegration(t *testing.T) {
	truthy := []string{"1", "t", "true"}
	v := strings.ToLower(os.Getenv("ENDURO_PP_INTEGRATION_TEST"))
	if !slices.Contains(truthy, v) {
		t.Skipf(
			"Set ENDURO_PP_INTEGRATION_TEST={%s} to run this test.",
			strings.Join(truthy, ","),
		)
	}

	ctx := context.Background()
	temporalServer := setUpTemporal(ctx, t)

	t.Run("Remove .DS_Store files", func(t *testing.T) {
		testTransfer := "small_with_ds_store"

		env := newTestEnv(t, defaultConfig())
		env.cfg.Temporal.Address = temporalServer.addr
		env.copyTestTransfer(testTransfer)
		env.startWorker(ctx)

		run, err := temporalServer.client.ExecuteWorkflow(
			ctx,
			temporalsdk_client.StartWorkflowOptions{
				TaskQueue:                env.cfg.Temporal.TaskQueue,
				WorkflowExecutionTimeout: 30 * time.Second,
			},
			workflow.NewPreprocessingWorkflow(env.testDir.Path()).Execute,
			&workflow.PreprocessingWorkflowParams{
				RelativePath: testTransfer,
			},
		)
		assert.NilError(t, err, "Workflow could not be started.")

		var result workflow.PreprocessingWorkflowResult
		run.Get(ctx, &result)

		assert.Equal(t, result, workflow.PreprocessingWorkflowResult{
			RelativePath: testTransfer,
		})
		assert.Assert(t, tfs.Equal(
			env.testDir.Path(),
			tfs.Expected(t,
				tfs.WithDir(testTransfer, tfs.WithMode(0o755),
					tfs.WithFile(
						"small.txt", "I am a small file.\n", tfs.WithMode(0o644),
					),
				),
			),
		))
	})
}
