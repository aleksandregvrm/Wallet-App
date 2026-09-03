package services

import (
	"context"
	"fmt"
	"go-task-wallet-service/api-gateway/internal/domain"
	infra "go-task-wallet-service/api-gateway/internal/infra/grpc"
	"go-task-wallet-service/api-gateway/internal/mapping"
	"go-task-wallet-service/shared/logging"
)

type AccountService struct {
	grpc *infra.GrpcHandler
	logging.Logger
}

func NewAccountService(grpc *infra.GrpcHandler) *AccountService {
	return &AccountService{
		grpc:   grpc,
		Logger: logging.NewInternalLogger(),
	}
}

func (s *AccountService) CreateAccount(ctx context.Context, account *domain.Account) error {
	pbCreateAccounts := mapping.ToCreateAccountProto(account)
	resp, err := s.grpc.WalletClient.CreateAccount(ctx, pbCreateAccounts)
	if err != nil {
		return fmt.Errorf("failed to create account for user %s: %w", account.OwnerUser, err)
	}

	created := mapping.ToAccountDomain(resp.GetAccount())
	if created != nil {
		*account = *created
	}

	return nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, account *domain.Account) error {
	return nil
}
