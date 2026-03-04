// client-card 是虚拟智能卡管理服务。
// 提供 IPC 接口供 pkcs11-mock 调用，以及 REST API 供前端管理。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/globaltrusts/client-card/configs"
	"github.com/globaltrusts/client-card/internal/api"
	"github.com/globaltrusts/client-card/internal/card"
	"github.com/globaltrusts/client-card/internal/card/cloud"
	"github.com/globaltrusts/client-card/internal/card/local"
	tpm2card "github.com/globaltrusts/client-card/internal/card/tpm2"
	"github.com/globaltrusts/client-card/internal/ipc"
	"github.com/globaltrusts/client-card/internal/storage"
	"github.com/globaltrusts/client-card/internal/tpm"
	"github.com/globaltrusts/client-card/pkg/pkcs11types"
)

// 版本信息，由 ldflags 在构建时注入。
var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径（默认使用用户数据目录）")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("client-card %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	initLogger(cfg.Log.Level)

	slog.Info("client-card 启动中",
		"version", Version,
		"commit", Commit,
		"api_addr", cfg.API.Addr(),
		"ipc_path", cfg.IPC.IPCPath(),
		"db_path", cfg.Database.Path,
	)

	// 初始化数据库
	db, err := storage.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("初始化数据库失败", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("数据库已连接", "path", cfg.Database.Path)

	// 初始化卡片管理器
	manager := card.NewManager()

	// 从数据库加载所有本地卡片，注册为 Slot
	if err := loadLocalSlots(manager, db); err != nil {
		slog.Warn("加载本地 Slot 失败", "error", err)
	}

	// 启动 IPC 服务
	ipcServer := ipc.NewServer(cfg.IPC.IPCPath())
	pkcsHandler := ipc.NewPKCSHandler(manager)
	pkcsHandler.Register(ipcServer)

	if err := ipcServer.Start(); err != nil {
		slog.Error("启动 IPC 服务失败", "error", err)
		os.Exit(1)
	}
	defer ipcServer.Stop()

	// 启动 REST API 服务
	apiServer := api.NewServer(&cfg.API, manager, db)
	if err := apiServer.Start(); err != nil {
		slog.Error("启动 REST API 服务失败", "error", err)
		os.Exit(1)
	}

	slog.Info("client-card 已就绪，等待连接...")

	// 等待退出信号
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("client-card 正在关闭...")

	// 优雅关闭 API 服务
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Stop(shutdownCtx); err != nil {
		slog.Warn("REST API 关闭超时", "error", err)
	}
}

// loadLocalSlots 从数据库加载所有本地卡片，注册到 Manager。
func loadLocalSlots(manager *card.Manager, db *storage.DB) error {
	ctx := context.Background()
	cardRepo := storage.NewCardRepo(db)
	certRepo := storage.NewCertRepo(db)

	cards, err := cardRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("查询卡片列表失败: %w", err)
	}

	// 初始化 TPM Provider（失败时降级，不影响本地 Slot）
	tpmProv, tpmErr := tpm.NewProvider()
	if tpmErr != nil {
		slog.Warn("TPM2 不可用，TPM2 卡片将无法加载", "error", tpmErr)
	}

	var slotID pkcs11types.SlotID = 1
	for _, c := range cards {
		switch c.SlotType {
		case storage.SlotTypeLocal:
			slot := local.New(slotID, c, certRepo)
			manager.RegisterSlot(slot)
			slog.Info("已注册本地 Slot", "slot_id", slotID, "card", c.CardName)
		case storage.SlotTypeTPM2:
			if tpmProv == nil {
				slog.Warn("跳过 TPM2 卡片（TPM2 不可用）", "card", c.CardName)
				continue
			}
			slot := tpm2card.New(slotID, c, certRepo, tpmProv)
			manager.RegisterSlot(slot)
			slog.Info("已注册 TPM2 Slot", "slot_id", slotID, "card", c.CardName, "platform", tpmProv.PlatformName())
		case storage.SlotTypeCloud:
			if c.CloudURL == "" || c.CloudCardUUID == "" {
				slog.Warn("跳过 Cloud 卡片（缺少 cloud_url 或 cloud_card_uuid）", "card", c.CardName)
				continue
			}
			slot := cloud.New(slotID, c)
			manager.RegisterSlot(slot)
			slog.Info("已注册 Cloud Slot", "slot_id", slotID, "card", c.CardName, "url", c.CloudURL)
		default:
			continue
		}
		slotID++
	}
	return nil
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

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})
	slog.SetDefault(slog.New(handler))
}
