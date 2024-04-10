package workflow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/preprocessing-moma/internal/activities"
	remove "github.com/artefactual-sdps/remove-files-activity"
	"go.artefactual.dev/tools/temporal"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
)

type PreprocessingWorkflowParams struct {
	RelativePath string
}

type PreprocessingWorkflowResult struct {
	RelativePath string
}

type PreprocessingWorkflow struct {
	sharedPath string
}

func NewPreprocessingWorkflow(sharedPath string) *PreprocessingWorkflow {
	return &PreprocessingWorkflow{
		sharedPath: sharedPath,
	}
}

func (w *PreprocessingWorkflow) Execute(
	ctx temporalsdk_workflow.Context,
	params *PreprocessingWorkflowParams,
) (r *PreprocessingWorkflowResult, e error) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// TODO Make the file path a part of the enduro config or check the configuration later.
	// A remove file works like a .gitignore file.
	removePath := "/home/preprocessing-moma/.config/.remove"

	// Remove hidden files.
	var removedPaths activities.RemovePathsResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), remove.RemoveFilesName, &remove.RemoveFilesParams{
		RemovePath: localPath,
		IgnorePath: removePath,
	}).Get(ctx, &removedPaths)
	if e != nil {
		return nil, e
	}

	// TODO: repackage MOMA SIP into a Bag.

	return &PreprocessingWorkflowResult{RelativePath: params.RelativePath}, e
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}
