package workflow_test

import (
	"path/filepath"
	"testing"

	remove "github.com/artefactual-sdps/remove-files-activity"
	"github.com/artefactual-sdps/temporal-activities/removefiles"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-sdps/preprocessing-moma/internal/config"
	"github.com/artefactual-sdps/preprocessing-moma/internal/workflow"
)

type PreprocessingTestSuite struct {
	suite.Suite
	temporalsdk_testsuite.WorkflowTestSuite

	env *temporalsdk_testsuite.TestWorkflowEnvironment

	// Each test creates its own temporary transfer directory.
	testDir string

	// Each test registers the workflow with a different name to avoid
	// duplicates.
	workflow *workflow.PreprocessingWorkflow
}

func (s *PreprocessingTestSuite) copyTestTransfer(src, relPath string) {
	err := cp.Copy(src, filepath.Join(s.testDir, relPath))
	if err != nil {
		s.T().Fatalf("copyTestTransfer: %v", err)
	}
}

func (s *PreprocessingTestSuite) SetupTest(
	cfg config.Configuration,
	transferPath, relPath string,
) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()

	// Register activities.
	s.env.RegisterActivityWithOptions(
		removefiles.NewActivity(cfg.RemoveFiles).Execute,
		temporalsdk_activity.RegisterOptions{Name: remove.RemoveFilesName},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(s.testDir)

	// Copy test files to testDir.
	s.copyTestTransfer(transferPath, relPath)
}

func (s *PreprocessingTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestPreprocessingWorkflow(t *testing.T) {
	suite.Run(t, new(PreprocessingTestSuite))
}

func (s *PreprocessingTestSuite) TestExecute() {
	relPath := "transfer"
	cfg := config.Configuration{
		RemoveFiles: removefiles.Config{RemoveNames: ".DS_Store"},
	}

	s.SetupTest(
		cfg,
		filepath.Join("testdata", "transfer_with_ds_store"),
		relPath,
	)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		removefiles.ActivityName,
		sessionCtx,
		&removefiles.ActivityParams{Path: filepath.Join(s.testDir, relPath)},
	).Return(
		&removefiles.ActivityResult{Count: 1}, nil,
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		&result,
		&workflow.PreprocessingWorkflowResult{RelativePath: relPath},
	)
}
