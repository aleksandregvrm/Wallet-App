package controllers

import (
	"encoding/json"
	"fmt"
	"go-task-wallet-service/api-gateway/internal/domain"
	internalHttp "go-task-wallet-service/api-gateway/internal/infra/http"
	"go-task-wallet-service/api-gateway/internal/utils"
	"go-task-wallet-service/shared/logging"
	"net/http"
)

type UserController struct {
	userSvc domain.UserService
	logging.Logger
}

func NewUserController(userSvc domain.UserService) *UserController {
	return &UserController{userSvc: userSvc, Logger: logging.NewInternalLogger()}
}

func (c *UserController) HandleUserRegister(w http.ResponseWriter, r *http.Request) {
	var req internalHttp.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	domainUser := domain.User{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}
	userAuth, err := c.userSvc.RegisterUser(r.Context(), &domainUser)

	if err != nil {
		errMsg := fmt.Sprintf("%v", err)
		errResponse := utils.ErrorResponse{
			Error:  "User Registration Failed",
			Reason: errMsg,
		}

		utils.WriteError(w, http.StatusBadRequest, errResponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.AuthenticateUserResponse{Status: "user registered", AccessToken: userAuth.AccessToken})
}

func (c *UserController) HandleUserLogin(w http.ResponseWriter, r *http.Request) {
	var req internalHttp.LoginUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	domainUser := domain.User{
		Username: req.Username,
		Password: req.Password,
	}

	resp, err := c.userSvc.LoginUser(r.Context(), &domainUser)
	if err != nil {
		errResponse := utils.ErrorResponse{
			Error:  "Login Failed",
			Reason: "Invalid Credentials",
		}
		utils.WriteError(w, http.StatusBadRequest, errResponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.AuthenticateUserResponse{Status: "user authenticated", AccessToken: resp.AccessToken})
}

func (c *UserController) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req internalHttp.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	if req.RefreshToken == "" {
		utils.WriteError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	token, err := c.userSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.RefreshTokenResponse{Token: token})
}
