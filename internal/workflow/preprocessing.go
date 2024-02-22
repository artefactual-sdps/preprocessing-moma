package workflow

import (
	"errors"
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

func (w *PreprocessingWorkflow) Execute(ctx temporalsdk_workflow.Context, params *PreprocessingWorkflowParams) (r *PreprocessingWorkflowResult, e error) {
	logger := temporalsdk_workflow.GetLogger(ctx)
	logger.Debug("PreprocessingWorkflow workflow running!", "params", params)

	if params == nil || params.RelativePath == "" {
		e = temporal.NewNonRetryableError(fmt.Errorf("error calling workflow with unexpected inputs"))
		return nil, e
	}

	var removePaths []string

	defer func() {
		var result activities.RemovePathsResult
		err := temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.RemovePathsName, &activities.RemovePathsParams{
			Paths: removePaths,
		}).Get(ctx, &result)
		e = errors.Join(e, err)

		logger.Debug("PreprocessingWorkflow workflow finished!", "result", r, "error", e)
	}()

	localPath := filepath.Join(w.sharedPath, filepath.Clean(params.RelativePath))

	// Extract package.
	var extractPackageRes activities.ExtractPackageResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.ExtractPackageName, &activities.ExtractPackageParams{
		Path: localPath,
	}).Get(ctx, &extractPackageRes)
	if e != nil {
		return nil, e
	}

	removePaths = append(removePaths, localPath, extractPackageRes.Path)

	// TODO Make the file path a part of the enduro config or check the configuration later
	// A succumb file works like a .gitignore file.
	succumbPath := "./.succumb"

	// Remove hidden files.
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), remove.RemoveSIPFilesName, &remove.RemoveSIPFilesParams{
		DestPath:   extractPackageRes.Path,
		ConfigPath: succumbPath,
	}).Get(ctx, nil)
	if e != nil {
		return nil, e
	}

	// Repackage SFA Sip into a Bag.
	var sipCreation activities.SipCreationResult
	e = temporalsdk_workflow.ExecuteActivity(withLocalActOpts(ctx), activities.SipCreationName, &activities.SipCreationParams{
		SipPath: extractPackageRes.Path,
	}).Get(ctx, &sipCreation)
	if e != nil {
		return nil, e
	}

	realPath, e := filepath.Rel(w.sharedPath, sipCreation.NewSipPath)
	if e != nil {
		return nil, e
	}

	return &PreprocessingWorkflowResult{RelativePath: realPath}, e
}

func withLocalActOpts(ctx temporalsdk_workflow.Context) temporalsdk_workflow.Context {
	return temporalsdk_workflow.WithActivityOptions(ctx, temporalsdk_workflow.ActivityOptions{
		ScheduleToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporalsdk_temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})
}
