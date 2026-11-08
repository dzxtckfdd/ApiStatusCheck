package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"userManager/internal/dto"
	"userManager/internal/repository"
	"userManager/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type Request struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *Request) validate() bool {
	if r.Login == "" || r.Password == "" {
		return false
	}

	return true
}

func (r *RefreshRequest) validate() bool {
	return r.RefreshToken != ""
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

func (uh *UserHandler) UserRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var request Request

	if err := decodeJSON(w, r, &request); err != nil {
		return
	}
	if !request.validate() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !request.validate() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userDto := dto.UserDto{
		Login:    request.Login,
		Password: request.Password,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	err := uh.userService.Create(ctx, userDto)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (uh *UserHandler) UserLoginHandler(w http.ResponseWriter, r *http.Request) {
	var request Request

	if err := decodeJSON(w, r, &request); err != nil {
		return
	}
	if !request.validate() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !request.validate() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userDto := dto.UserDto{
		Login:    request.Login,
		Password: request.Password,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	authResponse, err := uh.userService.Login(ctx, userDto)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  authResponse.AccessToken,
		"refresh_token": authResponse.RefreshToken,
	})
}

func (uh *UserHandler) UserRefreshHandler(w http.ResponseWriter, r *http.Request) {
	var request RefreshRequest

	if err := decodeJSON(w, r, &request); err != nil {
		return
	}
	if !request.validate() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	authResponse, err := uh.userService.Refresh(ctx, request.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse)
}
