package cron

// CronScheduler defines the interface for managing cron jobs.
type CronScheduler interface {
	Start()
	Stop()
}
