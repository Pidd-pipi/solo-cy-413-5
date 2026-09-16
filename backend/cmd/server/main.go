package main

import (
	"fmt"
	"github.com/blueship581/mindgarden/backend/internal/config"
	"github.com/blueship581/mindgarden/backend/internal/constants"
	"github.com/blueship581/mindgarden/backend/internal/handler"
	"github.com/blueship581/mindgarden/backend/internal/model"
	"github.com/blueship581/mindgarden/backend/internal/repository"
	"github.com/blueship581/mindgarden/backend/internal/router"
	"github.com/blueship581/mindgarden/backend/internal/service"
	"github.com/blueship581/mindgarden/backend/internal/util"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
	"os"
)

func main() {
	cfg, e := config.Load()
	if e != nil {
		panic(fmt.Errorf("load config: %w", e))
	}
	logger := util.NewLogger()
	db, e := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if e != nil {
		logger.Error("database connect failed", "error", e)
		os.Exit(1)
	}
	logger.Info(constants.LogDBConnected)
	if e = db.AutoMigrate(&model.User{}, &model.Mood{}, &model.Assessment{}, &model.Journal{}, &model.UserAssessment{},
		&model.AdjustmentPlan{}, &model.PlanVersion{}, &model.PlanDay{}, &model.PlanTask{}); e != nil {
		logger.Error("database migrate failed", "error", e)
		os.Exit(1)
	}
	logger.Info(constants.LogDBMigrated)
	ur := repository.NewUserRepository(db)
	mr := repository.NewMoodRepository(db)
	ar := repository.NewAssessmentRepository(db)
	jr := repository.NewJournalRepository(db)
	pr := repository.NewPlanRepository(db)
	pdr := repository.NewPlanDataRepository(db)
	if e = pr.EnsureIndexes(db); e != nil {
		logger.Error("plan index ensure failed", "error", e)
		os.Exit(1)
	}
	us := service.NewUserService(ur, logger)
	ms := service.NewMoodService(mr, logger)
	as := service.NewAssessmentService(ar, logger)
	js := service.NewJournalService(jr, logger)
	ps := service.NewPlanService(db, pr, pdr, logger)
	// 新情绪/日记/测评提交后，自动重算进行中计划的后续建议（钩子内部吞错，不影响原始写入）。
	ms.SetPlanHook(ps)
	js.SetPlanHook(ps)
	as.SetPlanHook(ps)
	if e = as.Seed(); e != nil {
		logger.Error("assessment seed failed", "error", e)
		os.Exit(1)
	}
	h := router.Handlers{
		User:       handler.NewUserHandler(us, as, logger, cfg.JWTSecret, cfg.JWTIssuer),
		Mood:       handler.NewMoodHandler(ms, logger),
		Assessment: handler.NewAssessmentHandler(as, logger),
		Journal:    handler.NewJournalHandler(js, logger),
		Plan:       handler.NewPlanHandler(ps, logger),
	}
	if e = router.New(cfg, h, logger).Run(":" + cfg.Port); e != nil {
		logger.Error("server stopped", "error", e)
		os.Exit(1)
	}
}

var _ *slog.Logger
