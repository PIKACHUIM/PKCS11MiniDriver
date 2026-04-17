package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/globaltrusts/server-card/internal/storage"
	"github.com/google/uuid"
)

// ---- 支付系统处理器 ----

// RechargeRequest 是充值请求体。
type RechargeRequest struct {
	AmountCents int64  `json:"amount_cents"` // 金额（分）
	Channel     string `json:"channel"`      // 支付渠道
}

func (s *Server) handleCreateRecharge(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req RechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.AmountCents <= 0 {
		writeError(w, http.StatusBadRequest, "充值金额必须大于0")
		return
	}
	if req.Channel == "" {
		writeError(w, http.StatusBadRequest, "支付渠道不能为空")
		return
	}

	notifyURL := fmt.Sprintf("%s/api/payment/callback/%s", s.cfg.API.BaseURL, req.Channel)
	order, payResp, err := s.paymentSvc.CreateRechargeOrder(r.Context(), claims.UserUUID, req.AmountCents, req.Channel, notifyURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"order":   order,
		"pay_url": payResp.PayURL,
		"qr_code": payResp.QRCode,
	})
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	page, pageSize := parsePagination(r)

	orders, total, err := s.paymentSvc.ListOrders(r.Context(), claims.UserUUID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"total":  total,
		"page":   page,
	})
}

func (s *Server) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	balance, err := s.paymentSvc.GetBalance(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, balance)
}

// RefundRequestBody 是退款请求体。
type RefundRequestBody struct {
	OrderNo string `json:"order_no"`
	Reason  string `json:"reason"`
}

func (s *Server) handleCreateRefund(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req RefundRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.OrderNo == "" {
		writeError(w, http.StatusBadRequest, "订单号不能为空")
		return
	}

	refund, err := s.paymentSvc.CreateRefund(r.Context(), claims.UserUUID, req.OrderNo, req.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 记录退款申请日志
	s.logRepo.Create(r.Context(), &storage.Log{ //nolint:errcheck
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("refund_request:%s", req.OrderNo),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "退款申请已提交，等待管理员审批",
		"refund":  refund,
	})
}

func (s *Server) handlePaymentCallback(w http.ResponseWriter, r *http.Request) {
	channel := r.PathValue("channel")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024)) // 限制 1MB
	if err != nil {
		writeError(w, http.StatusBadRequest, "读取请求体失败")
		return
	}

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	if err := s.paymentSvc.HandleCallback(r.Context(), channel, body, headers); err != nil {
		slog.Error("支付回调处理失败", "channel", channel, "error", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// 大多数支付平台期望返回 "success" 字符串
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success")) //nolint:errcheck
}

// ---- 支付插件管理处理器（管理员）----

func (s *Server) handleListPaymentPlugins(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(),
		`SELECT uuid, name, plugin_type, enabled, sort_weight, created_at, updated_at
		 FROM payment_plugins ORDER BY sort_weight DESC, created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var plugins []map[string]interface{}
	for rows.Next() {
		var p storage.PaymentPlugin
		var enabled int
		if err := rows.Scan(&p.UUID, &p.Name, &p.PluginType, &enabled, &p.SortWeight, &p.CreatedAt, &p.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		p.Enabled = enabled == 1
		// 不返回敏感配置
		plugins = append(plugins, map[string]interface{}{
			"uuid":        p.UUID,
			"name":        p.Name,
			"plugin_type": p.PluginType,
			"enabled":     p.Enabled,
			"sort_weight": p.SortWeight,
			"created_at":  p.CreatedAt,
			"updated_at":  p.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"plugins": plugins, "total": len(plugins)})
}

// CreatePaymentPluginRequest 是创建支付插件请求体。
type CreatePaymentPluginRequest struct {
	Name       string                 `json:"name"`
	PluginType string                 `json:"plugin_type"` // alipay/wechat/stripe/paypal
	Config     map[string]interface{} `json:"config"`      // API Key/Secret 等（将加密存储）
	Enabled    bool                   `json:"enabled"`
	SortWeight int                    `json:"sort_weight"`
}

func (s *Server) handleCreatePaymentPlugin(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Name == "" || req.PluginType == "" {
		writeError(w, http.StatusBadRequest, "插件名称和类型不能为空")
		return
	}

	// 加密存储配置
	var configEnc []byte
	if req.Config != nil {
		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			writeError(w, http.StatusBadRequest, "配置格式错误")
			return
		}
		// 使用 card service 的加密功能
		configEnc, err = s.cardSvc.EncryptData(configJSON)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "加密配置失败")
			return
		}
	}

	pluginUUID := uuid.New().String()
	now := time.Now()
	_, err := s.db.ExecContext(r.Context(),
		`INSERT INTO payment_plugins (uuid, name, plugin_type, config_enc, enabled, sort_weight, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		pluginUUID, req.Name, req.PluginType, configEnc, boolToIntLocal(req.Enabled), req.SortWeight, now, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"uuid":        pluginUUID,
		"name":        req.Name,
		"plugin_type": req.PluginType,
		"enabled":     req.Enabled,
		"sort_weight": req.SortWeight,
		"created_at":  now,
	})
}

// UpdatePaymentPluginRequest 是更新支付插件请求体。
type UpdatePaymentPluginRequest struct {
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	SortWeight int    `json:"sort_weight"`
}

func (s *Server) handleUpdatePaymentPlugin(w http.ResponseWriter, r *http.Request) {
	pluginUUID := r.PathValue("uuid")
	var req UpdatePaymentPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	_, err := s.db.ExecContext(r.Context(),
		`UPDATE payment_plugins SET name = ?, enabled = ?, sort_weight = ?, updated_at = ? WHERE uuid = ?`,
		req.Name, boolToIntLocal(req.Enabled), req.SortWeight, time.Now(), pluginUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "支付插件已更新"})
}

func (s *Server) handleDeletePaymentPlugin(w http.ResponseWriter, r *http.Request) {
	pluginUUID := r.PathValue("uuid")
	_, err := s.db.ExecContext(r.Context(), `DELETE FROM payment_plugins WHERE uuid = ?`, pluginUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "支付插件已删除"})
}

// ---- 退款审批处理器（管理员）----

func (s *Server) handleListRefunds(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	page, pageSize := parsePagination(r)
	offset := (page - 1) * pageSize

	query := `SELECT uuid, user_uuid, order_no, amount_cents, reason, status, approved_by, created_at, processed_at
		 FROM refund_requests WHERE 1=1`
	var args []interface{}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM refund_requests WHERE 1=1`
	if status != "" {
		countQuery += ` AND status = ?`
	}
	if err := s.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(r.Context(), query, append(args, pageSize, offset)...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var refunds []storage.RefundRequest
	for rows.Next() {
		var rr storage.RefundRequest
		var approvedBy sql.NullString
		var processedAt sql.NullTime
		if err := rows.Scan(&rr.UUID, &rr.UserUUID, &rr.OrderNo, &rr.AmountCents, &rr.Reason,
			&rr.Status, &approvedBy, &rr.CreatedAt, &processedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if approvedBy.Valid {
			rr.ApprovedBy = approvedBy.String
		}
		if processedAt.Valid {
			rr.ProcessedAt = &processedAt.Time
		}
		refunds = append(refunds, rr)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"refunds": refunds, "total": total, "page": page})
}

func (s *Server) handleApproveRefund(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	refundUUID := r.PathValue("uuid")

	// 查询退款工单
	var rr storage.RefundRequest
	err := s.db.QueryRowContext(r.Context(),
		`SELECT uuid, user_uuid, order_no, amount_cents, status FROM refund_requests WHERE uuid = ?`, refundUUID,
	).Scan(&rr.UUID, &rr.UserUUID, &rr.OrderNo, &rr.AmountCents, &rr.Status)
	if err != nil {
		writeError(w, http.StatusNotFound, "退款工单不存在")
		return
	}

	if rr.Status != "pending" {
		writeError(w, http.StatusBadRequest, "只能审批待处理的退款工单")
		return
	}

	now := time.Now()

	// 扣减用户余额
	_, err = s.db.ExecContext(r.Context(),
		`UPDATE user_balances SET available_cents = available_cents - ? WHERE user_uuid = ?`,
		rr.AmountCents, rr.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "扣减余额失败: "+err.Error())
		return
	}

	// 写入消费记录（负数金额表示退款）
	consumeUUID := uuid.New().String()
	_, err = s.db.ExecContext(r.Context(),
		`INSERT INTO consume_records (uuid, user_uuid, order_no, consume_type, amount_cents, remark, created_at)
		 VALUES (?, ?, ?, 'refund', ?, '退款审批通过', ?)`,
		consumeUUID, rr.UserUUID, rr.OrderNo, -rr.AmountCents, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "写入消费记录失败: "+err.Error())
		return
	}

	// 更新退款工单状态
	_, err = s.db.ExecContext(r.Context(),
		`UPDATE refund_requests SET status = 'paid', approved_by = ?, processed_at = ? WHERE uuid = ?`,
		claims.UserUUID, now, refundUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logRepo.Create(r.Context(), &storage.Log{ //nolint:errcheck
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("approve_refund:%s", refundUUID),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "退款已审批通过"})
}

func (s *Server) handleRejectRefund(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	refundUUID := r.PathValue("uuid")

	// 查询退款工单状态
	var status string
	err := s.db.QueryRowContext(r.Context(),
		`SELECT status FROM refund_requests WHERE uuid = ?`, refundUUID,
	).Scan(&status)
	if err != nil {
		writeError(w, http.StatusNotFound, "退款工单不存在")
		return
	}

	if status != "pending" {
		writeError(w, http.StatusBadRequest, "只能拒绝待处理的退款工单")
		return
	}

	now := time.Now()
	_, err = s.db.ExecContext(r.Context(),
		`UPDATE refund_requests SET status = 'failed', approved_by = ?, processed_at = ? WHERE uuid = ?`,
		claims.UserUUID, now, refundUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logRepo.Create(r.Context(), &storage.Log{ //nolint:errcheck
		UserUUID: claims.UserUUID,
		Action:   fmt.Sprintf("reject_refund:%s", refundUUID),
		IPAddr:   r.RemoteAddr,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "退款申请已拒绝"})
}

// boolToIntLocal 是本地 bool 转 int 工具函数。
func boolToIntLocal(b bool) int {
	if b {
		return 1
	}
	return 0
}
