package controllers

import (
	"encoding/json"
	"fmt"
	"go-task-wallet-service/api-gateway/internal/domain"
	internalHttp "go-task-wallet-service/api-gateway/internal/infra/http"
	"go-task-wallet-service/api-gateway/internal/utils"
	"net/http"
)

type AccountController struct {
	accountSvc domain.AccountService
}

func NewAccountController(accountSvc domain.AccountService) *AccountController {
	return &AccountController{accountSvc: accountSvc}
}

func (c *AccountController) HandleAccountCreate(w http.ResponseWriter, r *http.Request) {
	var domainAccount domain.Account
	if err := json.NewDecoder(r.Body).Decode(&domainAccount); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	domainAccount.OwnerUser = r.URL.Query().Get("userId")

	if err := utils.ValidateAccount(&domainAccount); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := c.accountSvc.CreateAccount(r.Context(), &domainAccount); err != nil {
		errMessage := fmt.Sprintf("%v", err)
		utils.WriteError(w, http.StatusBadRequest, utils.ErrorResponse{Error: "Account creation failed", Reason: errMessage})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.CreateAccountResponse{
		Status:    "account created",
		AccountID: domainAccount.ID,
		Balance:   domainAccount.Balance,
		Currency:  domainAccount.Currency,
	})
}

func (c *AccountController) HandleAccountUpdate(w http.ResponseWriter, r *http.Request) {
	var req internalHttp.UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	domainAccount := domain.Account{
		ID:        req.AccountID,
		OwnerUser: req.OwnerAccountId,
	}

	if err := c.accountSvc.UpdateAccount(r.Context(), &domainAccount); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "account update failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.UpdateAccountResponse{Status: "account updated"})
}
