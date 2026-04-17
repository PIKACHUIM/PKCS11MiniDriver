// Package ca - SM2 可用性公开接口。
package ca

// IsSM2Available 返回当前构建是否已启用 SM2 国密算法。
// 此函数对外暴露，便于 API/元数据层在运行时判定并向前端透出可用性。
func IsSM2Available() bool {
	return sm2Available()
}
