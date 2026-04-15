// Package template - 模板配置合法性校验。
package template

import (
	"fmt"

	"github.com/globaltrusts/server-card/internal/storage"
)

// Validate 校验密钥存储类型模板配置的合法性。
// 实现需求 25 中的所有业务规则。
func Validate(t *storage.KeyStorageTemplate) error {
	// 至少勾选一种存储方式
	if t.StorageMethods == 0 {
		return fmt.Errorf("至少需要选择一种存储方式")
	}

	if t.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	hasFileDownload := t.HasMethod(storage.StorageFileDownload)
	hasVirtualCard := t.HasMethod(storage.StorageVirtualCard)
	hasCloudCard := t.HasMethod(storage.StorageCloudCard)
	hasPhysicalCard := t.HasMethod(storage.StoragePhysicalCard)

	// 虚拟智能卡必须选择安全等级
	if hasVirtualCard {
		switch t.SecurityLevel {
		case storage.SecurityHigh, storage.SecurityMedium, storage.SecurityLow:
			// 合法
		default:
			return fmt.Errorf("勾选虚拟智能卡时必须选择安全等级（high/medium/low）")
		}
	}

	// 高安全性规则：不可导出、不可备份、不可重新导入
	if hasVirtualCard && t.SecurityLevel == storage.SecurityHigh {
		if t.CloudBackup {
			return fmt.Errorf("高安全性模式下不允许云端备份（TPM 内密钥不可导出）")
		}
		if t.AllowReimport {
			return fmt.Errorf("高安全性模式下不允许重新导入（密钥不可导出）")
		}
	}

	// 文件下载自动启用云端备份
	if hasFileDownload {
		t.CloudBackup = true
		t.AllowReissue = true
		t.MaxReissueCount = -1 // 无限次
	}

	// 未勾选文件下载时，才有"是否允许重新导入"选项
	// （文件下载本身就意味着密钥可以自由使用）
	if hasFileDownload {
		t.AllowReimport = false // 文件下载模式下此选项无意义
	}

	// 云端备份是重新下发的前提
	if t.AllowReissue && !t.CloudBackup {
		return fmt.Errorf("启用重新下发需要先启用云端私钥备份")
	}

	// 下发次数校验
	if t.AllowReissue && !hasFileDownload {
		if t.MaxReissueCount == 0 {
			return fmt.Errorf("启用重新下发时，最大下发次数不能为0（使用-1表示无限）")
		}
		if t.MaxReissueCount < -1 {
			return fmt.Errorf("最大下发次数不合法: %d", t.MaxReissueCount)
		}
	}

	// 实体/云端智能卡的云端备份选项校验
	if (hasCloudCard || hasPhysicalCard) && !hasVirtualCard && !hasFileDownload {
		// 仅有实体/云端卡时，云端备份是可选的，无需额外校验
	}

	return nil
}
