package handlers

import (
	"catalogService/internal/dto"
	"catalogService/internal/service"

	"encoding/json"
	"net/http"
)

type InternalHandler struct {
	internalService *service.InternalService
	secretKey       string
}

func NewInternalHandler(internalService *service.InternalService, secretKey string) *InternalHandler {
	return &InternalHandler{
		internalService: internalService,
		secretKey:       secretKey,
	}
}

func (h *InternalHandler) checkSecret(r *http.Request) bool {
	return r.Header.Get("X-Internal-Secret") == h.secretKey
}

func (ih *InternalHandler) GetActiveTargets(w http.ResponseWriter, r *http.Request) {
	if !ih.checkSecret(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	targets, err := ih.internalService.GetActiveResources(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(targets)
}

func (ih *InternalHandler) ProcessBatch(w http.ResponseWriter, r *http.Request) {
	if !ih.checkSecret(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req dto.SaveBatchRequest

	if err := decodeJSON(w, r, &req); err != nil {
		return
	}

	if err := ih.internalService.ProcessBatch(r.Context(), req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
