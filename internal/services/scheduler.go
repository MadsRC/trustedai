// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package services

import (
	"context"
	"log/slog"
	"time"
)

// Scheduler manages background jobs and periodic tasks
type Scheduler struct {
	logger         *slog.Logger
	costCalculator *CostCalculator
	stopChan       chan struct{}
	doneChan       chan struct{}
}

// SchedulerOption configures Scheduler behavior
type SchedulerOption func(*Scheduler)

// WithSchedulerLogger sets the logger for the scheduler
func WithSchedulerLogger(logger *slog.Logger) SchedulerOption {
	return func(s *Scheduler) {
		s.logger = logger
	}
}

// NewScheduler creates a new Scheduler instance
func NewScheduler(costCalculator *CostCalculator, options ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		logger:         slog.Default(),
		costCalculator: costCalculator,
		stopChan:       make(chan struct{}),
		doneChan:       make(chan struct{}),
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

// Start begins the scheduler's background operations
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("Starting background scheduler")

	go s.run(ctx)
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	s.logger.Info("Stopping background scheduler")
	close(s.stopChan)
	<-s.doneChan
	s.logger.Info("Background scheduler stopped")
}

// run executes the main scheduler loop
func (s *Scheduler) run(ctx context.Context) {
	defer close(s.doneChan)

	// Create tickers for different job types
	costCalculationTicker := time.NewTicker(5 * time.Minute) // Run cost calculation every 5 minutes
	billingTicker := time.NewTicker(1 * time.Hour)           // Run billing summaries every hour

	defer costCalculationTicker.Stop()
	defer billingTicker.Stop()

	s.logger.Info("Scheduler started",
		"costCalculationInterval", "5m",
		"billingInterval", "1h")

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler context cancelled")
			return

		case <-s.stopChan:
			s.logger.Info("Scheduler stop signal received")
			return

		case <-costCalculationTicker.C:
			s.runCostCalculation(ctx)

		case <-billingTicker.C:
			s.runBillingGeneration(ctx)
		}
	}
}

// runCostCalculation executes the cost calculation job
func (s *Scheduler) runCostCalculation(ctx context.Context) {
	s.logger.Info("Running scheduled cost calculation")
	start := time.Now()

	if err := s.costCalculator.ProcessUsageEvents(ctx); err != nil {
		s.logger.Error("Cost calculation job failed", "error", err, "duration", time.Since(start))
		return
	}

	s.logger.Info("Cost calculation job completed", "duration", time.Since(start))
}

// runBillingGeneration executes the billing summary generation job
func (s *Scheduler) runBillingGeneration(ctx context.Context) {
	s.logger.Info("Running scheduled billing summary generation")
	start := time.Now()

	// Generate daily summaries for yesterday
	yesterday := time.Now().AddDate(0, 0, -1)
	dailyPeriod := DailyPeriod{Date: yesterday}

	if err := s.costCalculator.GenerateBillingSummaries(ctx, dailyPeriod); err != nil {
		s.logger.Error("Daily billing summary generation failed", "error", err, "period", dailyPeriod.String())
	} else {
		s.logger.Info("Daily billing summary generation completed", "period", dailyPeriod.String())
	}

	// Generate monthly summaries for last month (on the 1st of each month)
	now := time.Now()
	if now.Day() == 1 && now.Hour() == 1 { // Run at 1 AM on the 1st of each month
		lastMonth := now.AddDate(0, -1, 0)
		monthlyPeriod := MonthlyPeriod{Year: lastMonth.Year(), Month: lastMonth.Month()}

		if err := s.costCalculator.GenerateBillingSummaries(ctx, monthlyPeriod); err != nil {
			s.logger.Error("Monthly billing summary generation failed", "error", err, "period", monthlyPeriod.String())
		} else {
			s.logger.Info("Monthly billing summary generation completed", "period", monthlyPeriod.String())
		}
	}

	s.logger.Info("Billing summary generation job completed", "duration", time.Since(start))
}
