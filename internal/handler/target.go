package handler

import (
	"encoding/json"
	"errors"
	"gowatch/internal/store"
	"net/http"
	"net/url"
)

type TargetHandler struct {
	store *store.Store
}

type createRequest struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

// 作成
func (h *TargetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// URL形式であるかのバリデーション（http & https含む）
	parsed, err := url.ParseRequestURI(req.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	target, err := h.store.AddTarget(r.Context(), req.URL, req.Name)
	if err != nil {
		if errors.Is(err, store.ErrorDuplicateURL) {
			http.Error(w, "duplicate url", http.StatusConflict)
			return
		}
		http.Error(w, "failed to add target", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(target)
}

// 全件取得
func (h *TargetHandler) Index(w http.ResponseWriter, r *http.Request) {
	targets, err := h.store.ListTargets(r.Context())
	if err != nil {
		http.Error(w, "failed to list targets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(targets)
}

// 削除
func (h *TargetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.store.DeleteTarget(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to delete target", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent) // 204
}

// コンストラクタ
func NewTargetHandler(store *store.Store) *TargetHandler {
	return &TargetHandler{store: store}
}
