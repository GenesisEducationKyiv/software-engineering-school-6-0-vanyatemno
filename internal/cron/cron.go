package cron

import (
	"context"
	"se-school/internal/config"
	repositorySvc "se-school/internal/services/repository"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	appCtx            context.Context
	cron              *cron.Cron
	repositoryService *repositorySvc.Service
}

// New creates a new Scheduler and registers all cron jobs based on the provided config.
func New(appCtx context.Context, cfg *config.Cron, repositoryService *repositorySvc.Service) *Scheduler {
	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(zap.NewStdLog(zap.L()))))

	s := &Scheduler{
		appCtx:            appCtx,
		cron:              c,
		repositoryService: repositoryService,
	}

	s.registerJobs(cfg)

	return s
}

// registerJobs adds all scheduled jobs to the cron scheduler.
func (s *Scheduler) registerJobs(cfg *config.Cron) {
	_, err := s.cron.AddFunc(cfg.RepoCheckSchedule, s.checkAllReposTagAndAlert)
	if err != nil {
		zap.L().Fatal("failed to register repo check cron job", zap.Error(err))
	}

	zap.L().Info("cron job registered", zap.String("schedule", cfg.RepoCheckSchedule), zap.String("job", "checkAllReposTagAndAlert"))
}

// checkAllReposTagAndAlert is the cron handler that invokes the repository service.
func (s *Scheduler) checkAllReposTagAndAlert() {
	zap.L().Info("cron: starting CheckAllReposTagAndAlert")

	err := s.repositoryService.CheckAllReposTagAndAlert(s.appCtx)
	if err != nil {
		zap.L().Error("cron: CheckAllReposTagAndAlert failed", zap.Error(err))
		return
	}

	zap.L().Info("cron: CheckAllReposTagAndAlert completed successfully")
}

// Start begins executing all registered cron jobs.
func (s *Scheduler) Start() {
	zap.L().Info("starting cron scheduler")
	s.cron.Start()
}

// Stop gracefully shuts down the cron scheduler, waiting for running jobs to finish.
func (s *Scheduler) Stop() {
	zap.L().Info("stopping cron scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	zap.L().Info("cron scheduler stopped")
}
