package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/sourcegraph/sourcegraph/lib/output"
	"github.com/sourcegraph/src-cli/internal/batches/executor"
)

type batchesLogEvent struct {
	Operation string `json:"operation"` // "PREPARING_DOCKER_IMAGES"

	Timestamp time.Time `json:"timestamp"`

	Status  string `json:"status"`            // "STARTED", "PROGRESS", "SUCCESS", "FAILURE"
	Message string `json:"message,omitempty"` // "70% done"
}

func logOperationStart(op, msg string) {
	logEvent(batchesLogEvent{Operation: op, Status: "STARTED", Message: msg})
}

func logOperationSuccess(op, msg string) {
	logEvent(batchesLogEvent{Operation: op, Status: "SUCCESS", Message: msg})
}

func logOperationFailure(op, msg string) {
	logEvent(batchesLogEvent{Operation: op, Status: "FAILURE", Message: msg})
}

func logOperationProgress(op, msg string) {
	logEvent(batchesLogEvent{Operation: op, Status: "PROGRESS", Message: msg})
}

func logEvent(e batchesLogEvent) {
	e.Timestamp = time.Now().UTC().Truncate(time.Millisecond)
	json.NewEncoder(os.Stdout).Encode(e)
}

type batchExecUI interface {
	ParsingBatchSpec()
	ParsingBatchSpecSuccess()

	ResolvingNamespace()
	ResolvingNamespaceSuccess(namespace string)

	PreparingContainerImages()
	PreparingContainerImagesProgress(percent float64)
	PreparingContainerImagesSuccess()

	DeterminingWorkspaceCreatorType()
	DeterminingWorkspaceCreatorTypeSuccess(creatorType string)

	ResolvingRepositories()
	ResolvingRepositoriesDone(unsupported, ignored, repos int)

	DeterminingWorkspaces()
	DeterminingWorkspacesSuccess(num int)

	CheckingCache()
	CheckingCacheSuccess(cachedSpecsFound int, tasksToExecute int)

	ExecutingTasks() func(ts []*executor.TaskStatus)
	ExecutingTasksSkippingErrors(err error)
	ExecutingTasksSuccess()

	LogFilesKept(files []string)

	UploadingChangesetSpecs(num int)
	UploadingChangesetSpecsProgress(done int)
	UploadingChangesetSpecsSuccess()

	CreatingBatchSpec()
	CreatingBatchSpecSuccess(url string)

	ApplyingBatchSpec()
	ApplyingBatchSpecSuccess(batchChangeURL string)
}

func batchCreatePending(out *output.Output, message string) output.Pending {
	return out.Pending(output.Line("", batchPendingColor, message))
}

func batchCompletePending(p output.Pending, message string) {
	p.Complete(output.Line(batchSuccessEmoji, batchSuccessColor, message))
}
