package handlers

import (
	"catalogService/internal/dto"
	"catalogService/internal/middleware"
	"catalogService/internal/repository"
	"catalogService/internal/service"
	"errors"
	"log"

	"encoding/json"
	"net/http"
)

type ClientHandler struct {
	resourceService *service.ResourceService
}

func NewClientHandler(resourceService *service.ResourceService) *ClientHandler {
	return &ClientHandler{resourceService: resourceService}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}

	return nil
}

func (ch *ClientHandler) CreateResource(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req dto.CreateResourceRequest

	if err := decodeJSON(w, r, &req); err != nil {
		return
	}

	resp, err := ch.resourceService.CreateResource(r.Context(), req, userID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Printf("CreateResource error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (ch *ClientHandler) GetUserResources(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resp, err := ch.resourceService.GetResourcesByUserId(r.Context(), userID)
	if err != nil {
		log.Printf("GetUserResources error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (ch *ClientHandler) GetPublicResources(w http.ResponseWriter, r *http.Request) {
	resp, err := ch.resourceService.GetPublicResources(r.Context())
	if err != nil {
		log.Printf("GetPublicResources error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (ch *ClientHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req dto.GetMetricsRequest

	if err := decodeJSON(w, r, &req); err != nil {
		return
	}

	resp, err := ch.resourceService.GetMetricsById(r.Context(), req, userID)
	if err != nil {
		if errors.Is(err, service.ErrAccessDenied) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if errors.Is(err, repository.ErrResourceNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		log.Printf("GetMetrics error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
