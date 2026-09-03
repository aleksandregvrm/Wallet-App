package integration_tests

import (
	"context"
	"testing"
	"time"

	types "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/repository"
	"go-task-wallet-service/wallet-service/tests/container"

	"github.com/google/uuid"
)

// accounts.user_id has a foreign key against users(id), so every account
// created in these tests needs a real row there first — wallet-service never
// creates users itself (that's auth-service's job), so we do it here directly.
func createUser(t *testing.T, env *container.TestEnv, userId string) {
	t.Helper()
	_, err := env.DB.Pool.Exec(context.Background(), `
		INSERT INTO users (id, name, email, username, password) VALUES ($1, $2, $3, $4, $5)
	`, userId, "Test User", userId+"@example.com", userId, "hashed-password")
	if err != nil {
		t.Fatalf("setup: failed to insert user %s: %v", userId, err)
	}
}

func insertAccount(t *testing.T, env *container.TestEnv, repo *repository.WalletRepository, currency string) *types.AccountModel {
	t.Helper()
	userId := uuid.NewString()
	createUser(t, env, userId)
	account, err := repo.InsertAccount(context.Background(), userId, currency)
	if err != nil {
		t.Fatalf("setup: failed to insert account: %v", err)
	}
	return account
}

func findTransaction(t *testing.T, repo *repository.WalletRepository, accountId, transactionId string) *types.TransactionModel {
	t.Helper()
	list, err := repo.ListTransactions(context.Background(), accountId, 0, 100)
	if err != nil {
		t.Fatalf("failed to list transactions for account %s: %v", accountId, err)
	}
	for i := range list {
		if list[i].ID == transactionId {
			return &list[i]
		}
	}
	return nil
}

func TestWalletRepository_InsertAccount_And_FindAccountById(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	createUser(t, env, userId)
	account, err := repo.InsertAccount(ctx, userId, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.UserID != userId || account.Currency != "USD" || account.Balance != 0 {
		t.Fatalf("unexpected inserted account: %+v", account)
	}

	found, err := repo.FindAccountById(ctx, account.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != account.ID || found.UserID != userId {
		t.Fatalf("unexpected found account: %+v", found)
	}
}

func TestWalletRepository_FindAccountsByUserId_ScopedToUser(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	otherUserId := uuid.NewString()
	createUser(t, env, userId)
	createUser(t, env, otherUserId)

	if _, err := repo.InsertAccount(ctx, userId, "USD"); err != nil {
		t.Fatalf("setup: failed to insert account: %v", err)
	}
	if _, err := repo.InsertAccount(ctx, userId, "EUR"); err != nil {
		t.Fatalf("setup: failed to insert account: %v", err)
	}
	if _, err := repo.InsertAccount(ctx, otherUserId, "USD"); err != nil {
		t.Fatalf("setup: failed to insert account: %v", err)
	}

	accounts, err := repo.FindAccountsByUserId(ctx, userId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts for the user, got: %d", len(accounts))
	}
	for _, a := range accounts {
		if a.UserID != userId {
			t.Fatalf("expected only accounts belonging to %s, got account for %s", userId, a.UserID)
		}
	}
}

func TestWalletRepository_CreatePendingTransaction_DuplicateIdIsNoop(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")

	transactionId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transactionId, "deposit", account.ID, account.ID, "USD", 500); err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}
	if err := repo.CreatePendingTransaction(ctx, transactionId, "deposit", account.ID, account.ID, "USD", 500); err != nil {
		t.Fatalf("unexpected error on duplicate create: %v", err)
	}

	list, err := repo.ListTransactions(ctx, account.ID, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 transaction row despite the duplicate create call, got: %d", len(list))
	}
}

func TestWalletRepository_DepositFunds_CreditsAccountAndCompletesTransaction(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")

	transactionId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transactionId, "deposit", account.ID, account.ID, "USD", 500); err != nil {
		t.Fatalf("setup: failed to create pending transaction: %v", err)
	}

	transaction, err := repo.DepositFunds(ctx, transactionId, account.ID, "USD", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transaction.Status != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be completed, got: %s", transaction.Status)
	}

	found, err := repo.FindAccountById(ctx, account.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Balance != 500 {
		t.Fatalf("expected balance 500, got: %d", found.Balance)
	}
}

func TestWalletRepository_DepositFunds_RedeliveryIsIdempotent(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")

	transactionId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transactionId, "deposit", account.ID, account.ID, "USD", 500); err != nil {
		t.Fatalf("setup: failed to create pending transaction: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, transactionId, account.ID, "USD", 500); err != nil {
		t.Fatalf("unexpected error on first deposit: %v", err)
	}

	// Simulating a redelivered Kafka message for the same transaction.
	if _, err := repo.DepositFunds(ctx, transactionId, account.ID, "USD", 500); err != nil {
		t.Fatalf("unexpected error on redelivered deposit: %v", err)
	}

	found, err := repo.FindAccountById(ctx, account.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Balance != 500 {
		t.Fatalf("expected the redelivered deposit not to double-credit the account, got balance: %d", found.Balance)
	}
}

func TestWalletRepository_WithdrawFunds_DebitsAccount(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")
	depositId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", account.ID, account.ID, "USD", 1000); err != nil {
		t.Fatalf("setup: failed to create pending deposit: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, depositId, account.ID, "USD", 1000); err != nil {
		t.Fatalf("setup: failed to deposit funds: %v", err)
	}

	withdrawalId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, withdrawalId, "withdrawal", account.ID, account.ID, "USD", 300); err != nil {
		t.Fatalf("setup: failed to create pending withdrawal: %v", err)
	}
	transaction, err := repo.WithdrawFunds(ctx, withdrawalId, account.ID, 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transaction.Status != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be completed, got: %s", transaction.Status)
	}

	found, err := repo.FindAccountById(ctx, account.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Balance != 700 {
		t.Fatalf("expected balance 700, got: %d", found.Balance)
	}
}

func TestWalletRepository_WithdrawFunds_InsufficientBalance_RollsBack(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")
	depositId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", account.ID, account.ID, "USD", 100); err != nil {
		t.Fatalf("setup: failed to create pending deposit: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, depositId, account.ID, "USD", 100); err != nil {
		t.Fatalf("setup: failed to deposit funds: %v", err)
	}

	withdrawalId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, withdrawalId, "withdrawal", account.ID, account.ID, "USD", 500); err != nil {
		t.Fatalf("setup: failed to create pending withdrawal: %v", err)
	}
	if _, err := repo.WithdrawFunds(ctx, withdrawalId, account.ID, 500); err == nil {
		t.Fatalf("expected an error for a withdrawal exceeding the available balance")
	}

	found, err := repo.FindAccountById(ctx, account.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Balance != 100 {
		t.Fatalf("expected the balance to remain unchanged after the rollback, got: %d", found.Balance)
	}

	pending := findTransaction(t, repo, account.ID, withdrawalId)
	if pending == nil {
		t.Fatalf("expected the withdrawal transaction row to still exist")
	}
	if pending.Status != types.TransactionStatusPending {
		t.Fatalf("expected the failed withdrawal to remain pending after rollback, got: %s", pending.Status)
	}
}

func TestWalletRepository_TransferFunds_MovesBetweenAccounts(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	from := insertAccount(t, env, repo, "USD")
	to := insertAccount(t, env, repo, "USD")
	depositId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", from.ID, from.ID, "USD", 1000); err != nil {
		t.Fatalf("setup: failed to create pending deposit: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, depositId, from.ID, "USD", 1000); err != nil {
		t.Fatalf("setup: failed to deposit funds: %v", err)
	}

	transferId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transferId, "transfer", from.ID, to.ID, "USD", 300); err != nil {
		t.Fatalf("setup: failed to create pending transfer: %v", err)
	}
	transaction, err := repo.TransferFunds(ctx, transferId, from.ID, to.ID, "USD", 300)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transaction.Status != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be completed, got: %s", transaction.Status)
	}

	fromFound, err := repo.FindAccountById(ctx, from.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toFound, err := repo.FindAccountById(ctx, to.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromFound.Balance != 700 {
		t.Fatalf("expected sender balance 700, got: %d", fromFound.Balance)
	}
	if toFound.Balance != 300 {
		t.Fatalf("expected receiver balance 300, got: %d", toFound.Balance)
	}
}

func TestWalletRepository_TransferFunds_InsufficientBalance_RollsBackBothAccounts(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	from := insertAccount(t, env, repo, "USD")
	to := insertAccount(t, env, repo, "USD")
	depositId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", from.ID, from.ID, "USD", 100); err != nil {
		t.Fatalf("setup: failed to create pending deposit: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, depositId, from.ID, "USD", 100); err != nil {
		t.Fatalf("setup: failed to deposit funds: %v", err)
	}

	transferId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transferId, "transfer", from.ID, to.ID, "USD", 500); err != nil {
		t.Fatalf("setup: failed to create pending transfer: %v", err)
	}
	if _, err := repo.TransferFunds(ctx, transferId, from.ID, to.ID, "USD", 500); err == nil {
		t.Fatalf("expected an error for a transfer exceeding the sender's balance")
	}

	fromFound, err := repo.FindAccountById(ctx, from.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toFound, err := repo.FindAccountById(ctx, to.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromFound.Balance != 100 || toFound.Balance != 0 {
		t.Fatalf("expected both balances to be rolled back unchanged, got from=%d to=%d", fromFound.Balance, toFound.Balance)
	}
}

func TestWalletRepository_TransferFunds_RedeliveryIsIdempotent(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	from := insertAccount(t, env, repo, "USD")
	to := insertAccount(t, env, repo, "USD")
	depositId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", from.ID, from.ID, "USD", 1000); err != nil {
		t.Fatalf("setup: failed to create pending deposit: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, depositId, from.ID, "USD", 1000); err != nil {
		t.Fatalf("setup: failed to deposit funds: %v", err)
	}

	transferId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transferId, "transfer", from.ID, to.ID, "USD", 300); err != nil {
		t.Fatalf("setup: failed to create pending transfer: %v", err)
	}
	if _, err := repo.TransferFunds(ctx, transferId, from.ID, to.ID, "USD", 300); err != nil {
		t.Fatalf("unexpected error on first transfer: %v", err)
	}
	// Simulating a redelivered Kafka message for the same transfer.
	if _, err := repo.TransferFunds(ctx, transferId, from.ID, to.ID, "USD", 300); err != nil {
		t.Fatalf("unexpected error on redelivered transfer: %v", err)
	}

	fromFound, err := repo.FindAccountById(ctx, from.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toFound, err := repo.FindAccountById(ctx, to.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromFound.Balance != 700 || toFound.Balance != 300 {
		t.Fatalf("expected the redelivered transfer not to move funds twice, got from=%d to=%d", fromFound.Balance, toFound.Balance)
	}
}

func TestWalletRepository_ListTransactions_OrdersByCreatedAtDescAndPaginates(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")

	var ids []string
	for i := 0; i < 3; i++ {
		id := uuid.NewString()
		if err := repo.CreatePendingTransaction(ctx, id, "deposit", account.ID, account.ID, "USD", 100); err != nil {
			t.Fatalf("setup: failed to create pending transaction %d: %v", i, err)
		}
		ids = append(ids, id)
	}

	page1, err := repo.ListTransactions(ctx, account.ID, 0, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected 2 transactions on the first page, got: %d", len(page1))
	}
	// Most recently created rows come first (ORDER BY created_at DESC).
	if page1[0].ID != ids[2] || page1[1].ID != ids[1] {
		t.Fatalf("unexpected ordering on page 1: %+v", page1)
	}

	page2, err := repo.ListTransactions(ctx, account.ID, 2, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page2) != 1 || page2[0].ID != ids[0] {
		t.Fatalf("unexpected page 2: %+v", page2)
	}
}

func TestWalletRepository_UpdateTransactionStatus_UpdatesRowAndReleasesIdempotencyLock(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")
	userId := account.UserID
	transactionId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, transactionId, "deposit", account.ID, account.ID, "USD", 100); err != nil {
		t.Fatalf("setup: failed to create pending transaction: %v", err)
	}

	idempotencyKey := userId + "_" + account.ID
	ok, err := env.Cache.TransactionIdempotencyCheck(idempotencyKey)
	if err != nil {
		t.Fatalf("setup: failed to acquire idempotency lock: %v", err)
	}
	if !ok {
		t.Fatalf("expected the idempotency lock to be free before any transaction is in flight")
	}
	if ok, err := env.Cache.TransactionIdempotencyCheck(idempotencyKey); err != nil || ok {
		t.Fatalf("expected the idempotency lock to be held, ok=%v err=%v", ok, err)
	}

	if err := repo.UpdateTransactionStatus(ctx, transactionId, userId, account.ID, types.TransactionStatusCompleted); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pending := findTransaction(t, repo, account.ID, transactionId)
	if pending == nil || pending.Status != types.TransactionStatusCompleted {
		t.Fatalf("expected the transaction to be marked completed, got: %+v", pending)
	}

	if ok, err := env.Cache.TransactionIdempotencyCheck(idempotencyKey); err != nil || !ok {
		t.Fatalf("expected UpdateTransactionStatus to release the idempotency lock, ok=%v err=%v", ok, err)
	}
}

func TestWalletRepository_UpdateTransactionStatus_NoRowFound_Errors(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	err := repo.UpdateTransactionStatus(ctx, uuid.NewString(), uuid.NewString(), uuid.NewString(), types.TransactionStatusCompleted)
	if err == nil {
		t.Fatalf("expected an error when no transaction row matches the id")
	}
}

func TestWalletRepository_OutboxInsert_BlocksConcurrentInFlightTransactionForSameAccount(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	userId := uuid.NewString()
	accountId := uuid.NewString()

	firstTxId := uuid.NewString()
	err := repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
		AggregateType: "transaction", AggregateId: firstTxId, EventType: "transaction.deposit.requested",
		Payload: map[string]interface{}{"ID": firstTxId}, UserId: userId, Topic: "wallet.events.v1", Partition_key: accountId,
	})
	if err != nil {
		t.Fatalf("unexpected error on first insert: %v", err)
	}

	secondTxId := uuid.NewString()
	err = repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
		AggregateType: "transaction", AggregateId: secondTxId, EventType: "transaction.withdrawal.requested",
		Payload: map[string]interface{}{"ID": secondTxId}, UserId: userId, Topic: "wallet.events.v1", Partition_key: accountId,
	})
	if err == nil {
		t.Fatalf("expected the second insert to be blocked while the first transaction is still in flight")
	}

	if err := repo.IdempotencyRelease(ctx, firstTxId, userId, accountId); err != nil {
		t.Fatalf("setup: failed to release idempotency: %v", err)
	}

	thirdTxId := uuid.NewString()
	err = repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
		AggregateType: "transaction", AggregateId: thirdTxId, EventType: "transaction.withdrawal.requested",
		Payload: map[string]interface{}{"ID": thirdTxId}, UserId: userId, Topic: "wallet.events.v1", Partition_key: accountId,
	})
	if err != nil {
		t.Fatalf("expected the insert to succeed once the lock is released: %v", err)
	}
}

func TestWalletRepository_Outbox_GetUpdateDelete_RoundTrip(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	aggregateId := uuid.NewString()
	err := repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
		AggregateType: "transaction", AggregateId: aggregateId, EventType: "transaction.deposit.requested",
		Payload: map[string]interface{}{"ID": aggregateId}, UserId: uuid.NewString(), Topic: "wallet.events.v1", Partition_key: uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("setup: failed to insert outbox row: %v", err)
	}

	row, err := repo.OutboxGet(ctx, aggregateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row == nil || row.Status != types.OutboxStatusPending {
		t.Fatalf("expected a pending outbox row, got: %+v", row)
	}

	if err := repo.OutboxUpdate(ctx, aggregateId, string(types.OutboxStatusPublished)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, err = repo.OutboxGet(ctx, aggregateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.Status != types.OutboxStatusPublished {
		t.Fatalf("expected a published outbox row, got: %+v", row)
	}

	if err := repo.OutboxDelete(ctx, aggregateId); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	row, err = repo.OutboxGet(ctx, aggregateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row != nil {
		t.Fatalf("expected the outbox row to be gone after delete, got: %+v", row)
	}
}

// This runs OutboxMarkPublishFailure against a real Postgres instance
// specifically because its CASE expression assigning into the outbox_status
// enum column needs explicit ::outbox_status casts — a bug there (assigning a
// plain text literal) type-checks fine in Go but only fails at the database,
// so a fake-repository unit test can never catch it.
func TestWalletRepository_OutboxMarkPublishFailure_RetriesThenExhausts(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	aggregateId := uuid.NewString()
	if err := repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
		AggregateType: "transaction", AggregateId: aggregateId, EventType: "transaction.deposit.requested",
		Payload: map[string]interface{}{"ID": aggregateId}, UserId: uuid.NewString(), Topic: "wallet.events.v1", Partition_key: uuid.NewString(),
	}); err != nil {
		t.Fatalf("setup: failed to insert outbox row: %v", err)
	}

	status, err := repo.OutboxMarkPublishFailure(ctx, aggregateId, "dial tcp: connection refused")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != string(types.OutboxStatusPending) {
		t.Fatalf("expected the row to still be pending (retryable), got: %s", status)
	}

	row, err := repo.OutboxGet(ctx, aggregateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.Status != types.OutboxStatusPending {
		t.Fatalf("expected a pending outbox row, got: %+v", row)
	}
	if row.ErrorMessage == nil || *row.ErrorMessage != "dial tcp: connection refused" {
		t.Fatalf("expected the error message to be recorded, got: %v", row.ErrorMessage)
	}

	var retryCount int
	var nextAttemptAt time.Time
	if err := env.DB.Pool.QueryRow(ctx, `SELECT retry_count, next_attempt_at FROM transaction_outbox WHERE aggregate_id = $1`, aggregateId).Scan(&retryCount, &nextAttemptAt); err != nil {
		t.Fatalf("unexpected error reading retry state: %v", err)
	}
	if retryCount != 1 {
		t.Fatalf("expected retry_count to be 1 after the first failure, got: %d", retryCount)
	}
	if !nextAttemptAt.After(time.Now()) {
		t.Fatalf("expected next_attempt_at to be pushed into the future by the backoff, got: %v", nextAttemptAt)
	}

	// Exhaust the remaining retries — the exact cap is an internal repository
	// detail, so drive it until it reports terminal rather than hardcoding a count.
	for i := 0; i < 10 && status != string(types.OutboxStatusFailed); i++ {
		status, err = repo.OutboxMarkPublishFailure(ctx, aggregateId, "dial tcp: connection refused")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if status != string(types.OutboxStatusFailed) {
		t.Fatalf("expected the row to eventually become terminally failed, got: %s", status)
	}

	row, err = repo.OutboxGet(ctx, aggregateId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.Status != types.OutboxStatusFailed {
		t.Fatalf("expected a failed outbox row, got: %+v", row)
	}
}

func TestWalletRepository_OutboxPendingGetBatch_OnlyPendingRespectingLimit(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	var aggregateIds []string
	for i := 0; i < 3; i++ {
		id := uuid.NewString()
		aggregateIds = append(aggregateIds, id)
		err := repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
			AggregateType: "transaction", AggregateId: id, EventType: "transaction.deposit.requested",
			Payload: map[string]interface{}{"ID": id}, UserId: uuid.NewString(), Topic: "wallet.events.v1", Partition_key: uuid.NewString(),
		})
		if err != nil {
			t.Fatalf("setup: failed to insert outbox row %d: %v", i, err)
		}
	}
	// Publishing one of them so it should no longer be considered pending.
	if err := repo.OutboxUpdate(ctx, aggregateIds[0], string(types.OutboxStatusPublished)); err != nil {
		t.Fatalf("setup: failed to publish outbox row: %v", err)
	}

	limited, err := repo.OutboxPendingGetBatch(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("expected the batch to respect the limit, got: %d", len(limited))
	}
	if limited[0].AggregateId == aggregateIds[0] {
		t.Fatalf("expected the published row not to be included in the pending batch")
	}

	rest, err := repo.OutboxPendingGetBatch(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rest) != 1 {
		t.Fatalf("expected 1 remaining pending row, got: %d", len(rest))
	}
	if rest[0].AggregateId == aggregateIds[0] {
		t.Fatalf("expected the published row not to be included in the pending batch")
	}
	if rest[0].AggregateId == limited[0].AggregateId {
		t.Fatalf("expected the second batch not to re-deliver a row already claimed by the first")
	}
}
