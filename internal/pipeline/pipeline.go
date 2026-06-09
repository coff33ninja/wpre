package pipeline

import (
	"fmt"

	"wpre/internal/state"
)

type Logger interface {
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
}

type Rollback interface {
	Save(string, interface{}) error
}

type Context struct {
	State    *state.PipelineState
	Config   interface{}
	Logger   Logger
	Rollback Rollback
	Data     *DataBus
	RunPS    func(args ...string) (string, error)
	RunPY    func(args ...string) (string, error)
}

type StageHandler func(ctx *Context) error

type Pipeline struct {
	handlers map[state.Stage]StageHandler
	order    []state.Stage
}

func New() *Pipeline {
	return &Pipeline{
		handlers: make(map[state.Stage]StageHandler),
		order: []state.Stage{
			state.StageBootstrap,
			state.StageProfileScan,
			state.StageBackupExisting,
			state.StageTempProfileCreate,
			state.StageDataHarvest,
			state.StageOneDriveExtract,
			state.StageAppCapture,
			state.StageNewProfileCreate,
			state.StageDataInject,
			state.StageFirstLoginInit,
			state.StageProfileCleanup,
			state.StageFinalValidation,
			state.StageComplete,
		},
	}
}

func (p *Pipeline) Register(stage state.Stage, handler StageHandler) {
	p.handlers[stage] = handler
}

func (p *Pipeline) Run(ctx *Context) error {
	for _, stage := range p.order {
		if err := p.runStage(ctx, stage); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) RunFrom(ctx *Context, start state.Stage) error {
	started := false
	for _, stage := range p.order {
		if stage == start {
			started = true
		}
		if !started {
			continue
		}
		if err := p.runStage(ctx, stage); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) runStage(ctx *Context, stage state.Stage) error {
	ctx.State.CurrentStage = stage

	handler, ok := p.handlers[stage]
	if !ok {
		ctx.Logger.Warn("[pipeline] no handler for stage: %s", stage)
		ctx.State.Advance(stage)
		return nil
	}

	snapshot := fmt.Sprintf("stage_%s", stage)
	if err := ctx.Rollback.Save(snapshot, ctx.Data); err != nil {
		ctx.Logger.Error("[pipeline] failed to save rollback point: %v", err)
	}

	ctx.Logger.Info("[pipeline] entering stage: %s", stage)

	if err := handler(ctx); err != nil {
		ctx.State.AddError(state.StageError{
			Stage:   stage,
			Message: err.Error(),
			Fatal:   true,
		})
		return fmt.Errorf("stage %s failed: %w", stage, err)
	}

	ctx.State.Advance(stage)
	ctx.Logger.Info("[pipeline] stage complete: %s", stage)
	return nil
}
