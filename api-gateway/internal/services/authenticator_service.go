package services

import (
	"fmt"
	"go-task-wallet-service/api-gateway/internal/utils"
	"go-task-wallet-service/shared/pkg/session"
)

func Authorize(token, userId string) *utils.ErrorResponse {
	refreshTokenClaims, err := session.ValidateAccessToken(token)
	if err != nil {
		return &utils.ErrorResponse{
			Error:  "Unauthorized",
			Reason: fmt.Sprintf("User with userId: %s, is not authorized on this route", userId),
		}
	}

	if refreshTokenClaims.UserId != userId {
		return &utils.ErrorResponse{
			Error:  "Unauthorized",
			Reason: fmt.Sprintf("User with userId: %s, is not authorized on this route", userId),
		}
	}
	// Authorization passed
	return nil
}
