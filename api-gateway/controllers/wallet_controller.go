package controllers

import (
	"encoding/json"
	"fmt"
	"go-task-wallet-service/api-gateway/internal/domain"
	internalHttp "go-task-wallet-service/api-gateway/internal/infra/http"
	"go-task-wallet-service/api-gateway/internal/utils"
	"log"
	"net/http"
)

type WalletController struct {
	walletService domain.WalletService
}

func NewWalletController(walletService domain.WalletService) *WalletController {
	return &WalletController{
		walletService: walletService,
	}
}

func (c *WalletController) HandleDeposit(w http.ResponseWriter, r *http.Request) {
	var depositFundsRequest internalHttp.DepositFundsRequest

	if err := json.NewDecoder(r.Body).Decode(&depositFundsRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	ownerUserId := r.URL.Query().Get("userId")
	if ownerUserId == "" {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request, missing userId")
		return
	}

	transaction, err := c.walletService.DepositFunds(r.Context(), ownerUserId, depositFundsRequest.AccountID, depositFundsRequest.Currency, depositFundsRequest.Amount)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to deposit funds, reason: %v", err)
		utils.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.DepositFundsResponse{
		Transaction: *transaction,
	})
}

func (c *WalletController) HandleWithdraw(w http.ResponseWriter, r *http.Request) {
	var withdrawFundsRequest internalHttp.WithdrawFundsRequest

	if err := json.NewDecoder(r.Body).Decode(&withdrawFundsRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	ownerUserId := r.URL.Query().Get("userId")
	if ownerUserId == "" {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request, missing userId")
		return
	}

	transaction, err := c.walletService.WithdrawFunds(r.Context(), ownerUserId, withdrawFundsRequest.AccountID, withdrawFundsRequest.Amount)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to withdraw funds, reason: %v", err)
		utils.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.WithdrawFundsResponse{
		Transaction: *transaction,
	})
}

func (c *WalletController) HandleTransferFunds(w http.ResponseWriter, r *http.Request) {
	var transferFundsRequest internalHttp.TransferFundsRequest

	if err := json.NewDecoder(r.Body).Decode(&transferFundsRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to parse JSON data")
		return
	}

	ownerUserId := r.URL.Query().Get("userId")

	if ownerUserId == "" {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request, missing userId")
		return
	}

	transaction, err := c.walletService.TransferFunds(r.Context(), ownerUserId, transferFundsRequest.FromAccountID, transferFundsRequest.ToAccountID, transferFundsRequest.Currency, transferFundsRequest.Amount)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to transfer funds, reason: %v", err)
		utils.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.TransferFundsResponse{
		Transaction: *transaction,
	})
}

func (c *WalletController) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("accountId")
	ownerUserId := r.URL.Query().Get("userId")

	if accountID == "" || ownerUserId == "" {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request, missing accountId or userId")
		return
	}

	balance, err := c.walletService.GetBalance(r.Context(), ownerUserId, accountID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to retrieve balance, reason: %v", err)
		utils.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(internalHttp.GetBalanceResponse{
		AccountID: balance.AccountID,
		Balance:   balance.Balance,
		Currency:  balance.Currency,
	})
}

func (c *WalletController) HandleListTransactions(w http.ResponseWriter, r *http.Request) {
	ownerAccount := r.URL.Query().Get("accountId")
	page := r.URL.Query().Get("page")
	pageSize := r.URL.Query().Get("pageSize")

	pageToInt, err := utils.ConvertStringToInt(page, 10, 16)
	if err != nil {
		log.Fatalf("Failed converting %s to integer", page)
	}

	pageSizeInt, err := utils.ConvertStringToInt(pageSize, 10, 8)

	// Validating fields which are key for querying the list of transactions
	if ownerAccount == "" || page == "" {
		utils.WriteError(w, http.StatusBadRequest, "Please provide required request data: owner accounts id, current page")
		return
	}

	listTransactionsResponse, err := c.walletService.ListTransactions(r.Context(), ownerAccount, int32(pageToInt), int8(pageSizeInt))
	if err != nil {
		errMsg := fmt.Sprintf("Failed to retrieve transactions, reason: %v", err)
		utils.WriteError(w, http.StatusBadRequest, errMsg)
		return
	}

	json.NewEncoder(w).Encode(internalHttp.ListTransactionsResponse{
		Transactions: listTransactionsResponse.Transactions,
		NextPage:     listTransactionsResponse.NextPageToken,
	})
}
