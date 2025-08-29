package base

import (
	"context"
	"fmt"

	"github.com/LumeraProtocol/supernode/v2/pkg/common/task"
	"github.com/LumeraProtocol/supernode/v2/pkg/common/task/state"
	"github.com/LumeraProtocol/supernode/v2/pkg/logtrace"
	"github.com/LumeraProtocol/supernode/v2/pkg/storage/files"
	"github.com/LumeraProtocol/supernode/v2/supernode/services/common"
)

// TaskCleanerFunc pointer to func that removes artefacts
type TaskCleanerFunc func()

// SuperNodeTask base "class" for Task
type SuperNodeTask struct {
	task.Task

	LogPrefix string
}

// RunHelper common code for Task runner
func (task *SuperNodeTask) RunHelper(ctx context.Context, clean TaskCleanerFunc) error {
	ctx = task.context(ctx)
	logtrace.Debug(ctx, "Start task", logtrace.Fields{})
	defer logtrace.Info(ctx, "Task canceled", logtrace.Fields{})
	defer task.Cancel()

	task.SetStatusNotifyFunc(func(status *state.Status) {
		logtrace.Debug(ctx, "States updated", logtrace.Fields{"status": status.String()})
	})

	defer clean()

	err := task.RunAction(ctx)

	// Update task status based on completion result
	if err != nil {
		task.UpdateStatus(common.StatusTaskCanceled)
	} else {
		task.UpdateStatus(common.StatusTaskCompleted)
	}

	return err
}

func (task *SuperNodeTask) context(ctx context.Context) context.Context {
	return logtrace.CtxWithCorrelationID(ctx, fmt.Sprintf("%s-%s", task.LogPrefix, task.ID()))
}

// RemoveFile removes file from FS (TODO: move to gonode.common)
func (task *SuperNodeTask) RemoveFile(file *files.File) {
	if file != nil {
		logtrace.Debug(context.Background(), "remove file", logtrace.Fields{"filename": file.Name()})
		if err := file.Remove(); err != nil {
			logtrace.Debug(context.Background(), "remove file failed", logtrace.Fields{logtrace.FieldError: err.Error()})
		}
	}
}

// NewSuperNodeTask returns a new Task instance.
func NewSuperNodeTask(logPrefix string) *SuperNodeTask {
	snt := &SuperNodeTask{
		Task:      task.New(common.StatusTaskStarted),
		LogPrefix: logPrefix,
	}

	return snt
}
