package cron

import (
	"context"
	"time"

	"se-school/internal/config"
	"se-school/internal/metrics"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// checkReposJob is the metric label identifying the repo-check cron job.
const checkReposJob = "check_repos"

type Scheduler struct {
	appCtx            context.Context
	cron              *cron.Cron
	repositoryService RepositoriesService
}

func New(appCtx context.Context, cfg *config.Cron, repositoryService RepositoriesService) *Scheduler {
	c := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(zap.NewStdLog(zap.L()))))

	s := &Scheduler{
		appCtx:            appCtx,
		cron:              c,
		repositoryService: repositoryService,
	}

	s.registerJobs(cfg)

	return s
}

func (s *Scheduler) registerJobs(cfg *config.Cron) {
	_, err := s.cron.AddFunc(cfg.RepoCheckSchedule, s.checkAllReposTagAndAlert)
	if err != nil {
		zap.L().Fatal("failed to register repo check cron job", zap.Error(err))
	}

	zap.L().Info("cron job registered", zap.String("schedule", cfg.RepoCheckSchedule), zap.String("job", "checkAllReposTagAndAlert"))
}

func (s *Scheduler) checkAllReposTagAndAlert() {
	zap.L().Info("cron: starting CheckAllReposTagAndAlert")

	start := time.Now()
	err := s.repositoryService.CheckAllReposTagAndAlert(s.appCtx)

	metrics.CronJobDuration.WithLabelValues(checkReposJob).Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.CronJobRunsTotal.WithLabelValues(checkReposJob, "error").Inc()
		zap.L().Error("cron: CheckAllReposTagAndAlert failed", zap.Error(err))
		return
	}

	metrics.CronJobRunsTotal.WithLabelValues(checkReposJob, "success").Inc()
	zap.L().Info("cron: CheckAllReposTagAndAlert completed successfully")
}

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
