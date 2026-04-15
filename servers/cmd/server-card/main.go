// servers 是云端智能卡管理服务。
// 提供 REST API 供 clients 的 Cloud Slot 调用。
// 私钥在服务端加密存储，签名/解密操作在服务端执行。
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/globaltrusts/server-card/configs"
	"github.com/globaltrusts/server-card/internal/acme"
	"github.com/globaltrusts/server-card/internal/api"
	"github.com/globaltrusts/server-card/internal/auth"
	"github.com/globaltrusts/server-card/internal/ca"
	"github.com/globaltrusts/server-card/internal/card"
	"github.com/globaltrusts/server-card/internal/ct"
	"github.com/globaltrusts/server-card/internal/issuance"
	"github.com/globaltrusts/server-card/internal/payment"
	"github.com/globaltrusts/server-card/internal/revocation"
	"github.com/globaltrusts/server-card/internal/storage"
	"github.com/globaltrusts/server-card/internal/template"
	"github.com/globaltrusts/server-card/internal/verification"
	"github.com/globaltrusts/server-card/internal/workflow"
)

func main() {
	// 加载配置
	cfg := configs.Load()

	// 初始化日志
	initLogger(cfg.Log.Level)

	slog.Info("servers 启动中", "addr", cfg.API.Addr(), "db", cfg.Database.Path)

	// 初始化数据库
	db, err := storage.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 初始化服务端主密钥（生产环境应从 HSM 或密钥管理服务获取）
	masterKey, err := loadOrGenerateMasterKey(cfg)
	if err != nil {
		slog.Error("初始化服务端主密钥失败", "error", err)
		os.Exit(1)
	}

	// 初始化各层
	userRepo := storage.NewUserRepo(db)
	cardRepo := storage.NewCardRepo(db)
	certRepo := storage.NewCertRepo(db)
	logRepo := storage.NewLogRepo(db)

	jwtMgr, err := auth.NewManager(cfg.JWT.Secret, cfg.JWT.ExpiryHours)
	if err != nil {
		slog.Error("初始化 JWT Manager 失败", "error", err)
		os.Exit(1)
	}
	cardSvc := card.NewService(cardRepo, certRepo, masterKey)

	// 初始化 CA 管理服务
	caSvc := ca.NewService(db, masterKey)

	// 初始化证书颁发模板服务
	issuanceSvc := issuance.NewService(db)

	// 初始化验证服务
	verifySvc := verification.NewService(db)

	// 初始化工作流服务
	workflowSvc := workflow.NewService(db)

	// 初始化支付系统
	paymentRegistry := payment.NewRegistry()
	paymentSvc := payment.NewService(db, paymentRegistry)

	// 初始化模板服务
	tmplSvc := template.NewService(db)

	// 初始化吊销服务
	revocationSvc := revocation.NewService(db, caSvc)

	// 初始化 CT 服务
	ctSvc := ct.NewService(db)

	// 初始化 ACME 服务
	acmeSvc := acme.NewService(db)

	// 启动定时任务：关闭超时未支付订单
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if n, err := paymentSvc.CloseExpiredOrders(context.Background()); err != nil {
				slog.Error("关闭超时订单失败", "error", err)
			} else if n > 0 {
				slog.Info("已关闭超时订单", "count", n)
			}
		}
	}()

	// 启动 API 服务
	svcs := &api.Services{
		CardSvc:       cardSvc,
		CASvc:         caSvc,
		IssuanceSvc:   issuanceSvc,
		VerifySvc:     verifySvc,
		WorkflowSvc:   workflowSvc,
		PaymentSvc:    paymentSvc,
		TmplSvc:       tmplSvc,
		RevocationSvc: revocationSvc,
		CTSvc:         ctSvc,
		ACMESvc:       acmeSvc,
	}
	apiServer := api.NewServer(cfg, db, jwtMgr, svcs, userRepo, logRepo)
	if err := apiServer.Start(); err != nil {
		slog.Error("启动 API 服务失败", "error", err)
		os.Exit(1)
	}

	slog.Info("servers 已就绪", "addr", cfg.API.Addr())

	// 等待退出信号
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 启动 CRL 定时刷新（使用信号上下文，优雅退出时自动停止）
	revocationSvc.StartCRLRefreshLoop(ctx)

	<-ctx.Done()

	slog.Info("servers 正在关闭...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Stop(shutdownCtx); err != nil {
		slog.Warn("API 服务关闭超时", "error", err)
	}
}

// loadOrGenerateMasterKey 加载或生成服务端主密钥。
// 主密钥持久化到文件（生产环境应使用 HSM 或密钥管理服务）。
func loadOrGenerateMasterKey(cfg *configs.Config) ([]byte, error) {
	keyPath := cfg.Database.Path + ".masterkey"

	// 尝试读取已有密钥
	if data, err := os.ReadFile(keyPath); err == nil && len(data) == 32 {
		slog.Info("已加载服务端主密钥")
		return data, nil
	}

	// 生成新密钥
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("生成主密钥失败: %w", err)
	}

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("保存主密钥失败: %w", err)
	}

	slog.Info("已生成新的服务端主密钥", "path", keyPath)
	return key, nil
}

// initLogger 初始化结构化日志。
func initLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}
