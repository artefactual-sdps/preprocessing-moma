package workflow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/artefactual-sdps/temporal-activities/removefiles"
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

	// Remove unwanted files.
	var removeFilesResult removefiles.ActivityResult
	e = temporalsdk_workflow.ExecuteActivity(
		withLocalActOpts(ctx),
		removefiles.ActivityName,
		&removefiles.ActivityParams{Path: localPath},
	).Get(ctx, &removeFilesResult)
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
