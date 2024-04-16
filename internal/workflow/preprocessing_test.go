package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/artefactual-sdps/preprocessing-moma/internal/workflow"
	remove "github.com/artefactual-sdps/remove-files-activity"
	cp "github.com/otiai10/copy"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"
)

type PreprocessingTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment

	// Each test creates its own temporary transfer directory.
	testDir string

	// Each test registers the workflow with a different name to avoid
	// duplicates.
	workflow *workflow.PreprocessingWorkflow
}

func (s *PreprocessingTestSuite) createRemoveFile(contents string) {
	path := filepath.Join(s.testDir, ".remove")
	err := os.WriteFile(path, []byte(contents), 0o600)
	if err != nil {
		s.T().Fatalf("createRemoveFile: %v", err)
	}
}

func (s *PreprocessingTestSuite) copyTestTransfer(src, relPath string) {
	err := cp.Copy(src, filepath.Join(s.testDir, relPath))
	if err != nil {
		s.T().Fatalf("copyTestTransfer: %v", err)
	}
}

func (s *PreprocessingTestSuite) SetupTest(transferPath, relPath string) {
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
	s.testDir = s.T().TempDir()

	// Register activities.
	s.env.RegisterActivityWithOptions(
		remove.NewRemoveFilesActivity().Execute,
		temporalsdk_activity.RegisterOptions{Name: remove.RemoveFilesName},
	)

	s.workflow = workflow.NewPreprocessingWorkflow(s.testDir)

	// Create test files in testDir.
	s.createRemoveFile(`.DS_Store
`)
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
	s.SetupTest(
		filepath.Join("testdata", "transfer_with_ds_store"),
		relPath,
	)

	// Mock activities.
	sessionCtx := mock.AnythingOfType("*context.timerCtx")
	s.env.OnActivity(
		remove.RemoveFilesName,
		sessionCtx,
		&remove.RemoveFilesParams{
			RemovePath: filepath.Join(s.testDir, relPath),
			IgnorePath: filepath.Join(s.testDir, ".remove"),
		},
	).Return(
		&remove.RemoveFilesResult{Removed: []string{".DS_Store"}},
	)

	s.env.ExecuteWorkflow(
		s.workflow.Execute,
		&workflow.PreprocessingWorkflowParams{RelativePath: relPath},
	)

	// Assert results.
	s.True(s.env.IsWorkflowCompleted())

	var result workflow.PreprocessingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.Equal(
		result,
		&workflow.PreprocessingWorkflowResult{RelativePath: relPath},
	)
}
