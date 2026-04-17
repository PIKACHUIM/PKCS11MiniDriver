package api

import (
	"encoding/json"
	"net/http"

	"github.com/globaltrusts/server-card/internal/storage"
)

// ---- 证书订单与申请处理器 ----

func (s *Server) handleCreateCertOrder(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var order storage.CertOrder
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	order.UserUUID = claims.UserUUID
	if err := s.workflowSvc.CreateOrder(r.Context(), &order); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleListCertOrders(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	page, pageSize := parsePagination(r)
	orders, total, err := s.workflowSvc.ListOrders(r.Context(), claims.UserUUID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": orders, "total": total})
}

func (s *Server) handleGetCertOrder(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	orderUUID := r.PathValue("uuid")
	order, err := s.workflowSvc.GetOrder(r.Context(), orderUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	// 非管理员只能查看自己的订单
	if order.UserUUID != claims.UserUUID && claims.Role != "admin" && claims.Role != "super_admin" {
		writeError(w, http.StatusForbidden, "无权查看此订单")
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handlePayCertOrder(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	orderUUID := r.PathValue("uuid")
	if err := s.workflowSvc.PayOrder(r.Context(), orderUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "订单已支付"})
}

func (s *Server) handleCancelCertOrder(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	orderUUID := r.PathValue("uuid")
	if err := s.workflowSvc.CancelOrder(r.Context(), orderUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "订单已取消"})
}

func (s *Server) handleCreateCertApplication(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var app storage.CertApplication
	if err := json.NewDecoder(r.Body).Decode(&app); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	app.UserUUID = claims.UserUUID
	if err := s.workflowSvc.CreateApplication(r.Context(), &app); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, app)
}

func (s *Server) handleListCertApplications(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	userUUID := claims.UserUUID
	if claims.Role == "admin" {
		userUUID = "" // 管理员查看所有
	}
	statusFilter := r.URL.Query().Get("status")
	page, pageSize := parsePagination(r)
	apps, total, err := s.workflowSvc.ListApplications(r.Context(), userUUID, statusFilter, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"applications": apps, "total": total})
}

func (s *Server) handleApproveCertApplication(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	appUUID := r.PathValue("uuid")
	if err := s.workflowSvc.ApproveApplication(r.Context(), appUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书申请已审批通过"})
}

func (s *Server) handleRejectCertApplication(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	appUUID := r.PathValue("uuid")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := s.workflowSvc.RejectApplication(r.Context(), appUUID, claims.UserUUID, req.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书申请已拒绝"})
}
