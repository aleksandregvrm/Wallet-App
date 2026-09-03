package middlewares

import (
	"fmt"
	"go-task-wallet-service/api-gateway/internal/utils"
	"go-task-wallet-service/shared/pkg/session"
	"net/http"
	"strings"
)

// Middlewares to check customer authentication, authorization

func AuthorizeUser(handler http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")

		errResponse := utils.ErrorResponse{
			Error:  "Unauthorized",
			Reason: "Unauthorized to access this route",
		}

		if authorization == "" {
			utils.WriteError(w, http.StatusUnauthorized, errResponse)
			return
		}

		if !strings.HasPrefix(authorization, "Bearer ") {
			utils.WriteError(w, http.StatusUnauthorized, errResponse)
			return
		}

		token := strings.TrimPrefix(authorization, "Bearer ")

		query := r.URL.Query()

		userId := query.Get("userId")

		if userId == "" {
			utils.WriteError(w, http.StatusUnauthorized, errResponse)
			return
		}

		result := authorize(token, userId)

		if result != nil {
			utils.WriteError(w, http.StatusUnauthorized, errResponse)
			return
		}

		// If all checks pass delegate request to handler, which is our controller
		handler(w, r)
	}
}

func authorize(token, userId string) *utils.ErrorResponse {
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
