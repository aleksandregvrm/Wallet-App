package repository

import (
	"context"
	"errors"
	"fmt"
	"go-task-wallet-service/shared/cache"
	"go-task-wallet-service/shared/db"
	"go-task-wallet-service/shared/env"
	"go-task-wallet-service/shared/logging"
	models "go-task-wallet-service/shared/pkg/models"
	types "go-task-wallet-service/shared/pkg/models"
	"go-task-wallet-service/wallet-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WalletRepository struct {
	db *db.DBClient
	ch cache.Cache
	logging.Logger
}

func NewWalletRepository(db *db.DBClient, ch cache.Cache) *WalletRepository {
	return &WalletRepository{
		db:     db,
		ch:     ch,
		Logger: logging.NewInternalLogger(),
	}
}

// Wallet related operations
func (r *WalletRepository) InsertAccount(ctx context.Context, userId, currency string) (*models.AccountModel, error) {
	var account models.AccountModel

	err := r.db.Pool.QueryRow(ctx, `
		INSERT INTO accounts (id, user_id, currency)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, balance, currency, created_at, updated_at
	`, uuid.NewString(), userId, currency).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		r.Logger.LogWarn(ctx, "Failed to insert account for user %s: %v", userId, err)
		return nil, fmt.Errorf("failed to insert account for user %s: %w", userId, err)
	}

	return &account, nil
}

func (r *WalletRepository) FindAccountById(ctx context.Context, accountId string) (*models.AccountModel, error) {
	var account models.AccountModel

	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, user_id, balance, currency, created_at, updated_at FROM accounts WHERE id = $1
	`, accountId).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to find account %s: %w", accountId, err)
	}

	return &account, nil
}

func (r *WalletRepository) FindAccountsByUserId(ctx context.Context, userId string) ([]types.AccountModel, error) {
	query := `
        SELECT id, user_id, balance, currency, created_at, updated_at FROM accounts WHERE user_id = $1
    `

	rows, err := r.db.Query(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []types.AccountModel

	for rows.Next() {
		var account types.AccountModel
		if err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.Balance,
			&account.Currency,
			&account.CreatedAt,
			&account.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func (r *WalletRepository) UpdateAccount(ctx context.Context, fromAccountId, toAccountId string) (*models.AccountModel, error) {
	return nil, nil
}

func (r *WalletRepository) GetOrStoreAccountBalance(ctx context.Context, accountId string, amount int64) (int64, error) {
	balance, err := r.ch.GetOrStoreAccountBalance(accountId, amount)
	if err != nil {
		r.Logger.LogInfo(ctx, "Failed to get or store balance in cache for account %s: %v", accountId, err)
		return amount, nil
	}

	return balance, nil
}

func (r *WalletRepository) InvalidateAccountBalance(ctx context.Context, accountId string) error {
	if err := r.ch.InvalidateAccountBalance(accountId); err != nil {
		r.Logger.LogError(ctx, "Failed to invalidate cached balance for account %s: %v", accountId, err)
		return err
	}

	return nil
}

// Actual reconciliation query which calculates balance based on the spend funds received funds
// Works similarly to blockchain although we still have balance column present with the accounts for read purposes only
// The DB isolation level we have chosen is READ COMMITTED since our application is write heavy we will use the postgreSql-s default isolation level
// This isolation level is noticeably faster than SERIALIZABLE for example, but it does not guarantee that the balance will be correct if there are concurrent transactions. However, since we are using a single transaction to update the balance, we can ensure that the balance is always correct.
func (r *WalletRepository) reconcileBalance(ctx context.Context, tx pgx.Tx, accountId string) (int64, error) {
	if _, err := tx.Exec(ctx, `SELECT id FROM accounts WHERE id = $1 FOR UPDATE`, accountId); err != nil {
		return 0, err
	}

	var balance int64
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			CASE
				WHEN transaction_type = 'deposit' THEN amount
				WHEN transaction_type = 'withdrawal' THEN -amount
				WHEN transaction_type = 'transfer' AND to_account = $1 THEN amount
				WHEN transaction_type = 'transfer' AND from_account = $1 THEN -amount
				ELSE 0
			END
		), 0)
		FROM transactions
		WHERE (from_account = $1 OR to_account = $1) AND status = 'completed'
	`, accountId).Scan(&balance)
	if err != nil {
		return 0, err
	}

	if balance < 0 {
		return 0, nil
	}

	cmd, err := tx.Exec(ctx, `UPDATE accounts SET balance = $2, updated_at = now() WHERE id = $1`, accountId, balance)
	if err != nil {
		return 0, err
	}

	return cmd.RowsAffected(), nil
}

// TransferFunds owns the full transaction lifecycle internally — it commits
// on success and rolls back on any error before returning; the caller never
// sees a transaction handle. Mirrors DepositFunds/WithdrawFunds: the pending
// ledger row must already exist (see CreatePendingTransaction) and this only
// flips it to completed, skipping the reconcile on redelivery.
func (r *WalletRepository) TransferFunds(ctx context.Context, transactionId, fromAccountId, toAccountId, currency string, amount int64) (*models.TransactionModel, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive, got %v", amount)
	}

	// Mix matching the account ids to ensure that the reconcileBalance is always called in the same order for both accounts, preventing deadlocks.
	firstId, secondId := fromAccountId, toAccountId
	if secondId < firstId {
		firstId, secondId = secondId, firstId
	}

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op once Commit has succeeded

	// Flips the pre-created pending row to completed. A conflict here (0 rows) means
	// a prior (possibly redelivered) attempt already completed it — skip the reconcile.
	update, err := tx.Exec(ctx, `
		UPDATE transactions SET status = 'completed', updated_at = now()
		WHERE id = $1 AND status = 'pending'
	`, transactionId)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if update.RowsAffected() == 1 {
		firstRows, err := r.reconcileBalance(ctx, tx, firstId)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile balance for account %s: %w", firstId, err)
		}
		if firstRows != 1 {
			return nil, fmt.Errorf("account %s not found or insufficient balance", firstId)
		}

		secondRows, err := r.reconcileBalance(ctx, tx, secondId)
		if err != nil {
			return nil, fmt.Errorf("failed to reconcile balance for account %s: %w", secondId, err)
		}
		if secondRows != 1 {
			return nil, fmt.Errorf("account %s not found or insufficient balance", secondId)
		}
	}

	var transaction models.TransactionModel
	if err := tx.QueryRow(ctx, `
		SELECT id, from_account, to_account, amount, currency, status, created_at, updated_at
		FROM transactions WHERE id = $1
	`, transactionId).Scan(
		&transaction.ID,
		&transaction.FromAccount,
		&transaction.ToAccount,
		&transaction.Amount,
		&transaction.Currency,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to read transaction %s: %w", transactionId, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transfer from %s to %s: %w", fromAccountId, toAccountId, err)
	}

	return &transaction, nil
}

// In any case the transaction is. being written in the db and then the final status is applied after processing is either failed or completed
func (r *WalletRepository) CreatePendingTransaction(ctx context.Context, transactionId, transactionType, fromAccount, toAccount, currency string, amount int64) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Explicit check in case the same account is not making a transfer
	// Deadlock preventive measure to lock the smaller id first in case both account are making interchanging transfers
	// at the same time
	if fromAccount != toAccount {
		firstId, secondId := fromAccount, toAccount
		if secondId < firstId {
			firstId, secondId = secondId, firstId
		}
		for _, id := range []string{firstId, secondId} {
			if _, err := tx.Exec(ctx, `SELECT id FROM accounts WHERE id = $1 FOR UPDATE`, id); err != nil {
				r.Logger.LogWarn(ctx, "Failed to create pending transaction %s: %v", transactionId, err)
				return fmt.Errorf("failed to create pending transaction %s: %w", transactionId, err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO transactions (id, from_account, to_account, amount, currency, status, transaction_type)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6)
		ON CONFLICT (id) DO NOTHING
	`, transactionId, fromAccount, toAccount, amount, currency, transactionType); err != nil {
		r.Logger.LogWarn(ctx, "Failed to create pending transaction %s: %v", transactionId, err)
		return fmt.Errorf("failed to create pending transaction %s: %w", transactionId, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit pending transaction %s: %w", transactionId, err)
	}

	return nil
}

// DepositFunds owns the full transaction lifecycle internally — it commits
// on success and rolls back on any error before returning; the caller never
// sees a transaction handle.
func (r *WalletRepository) DepositFunds(ctx context.Context, transactionId, accountId, currency string, amount int64) (*models.TransactionModel, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // no-op once Commit has succeeded

	// Flips the pre-created pending row to completed. A conflict here (0 rows) means
	// a prior (possibly redelivered) attempt already completed it — skip the credit.
	update, err := tx.Exec(ctx, `
		UPDATE transactions SET status = 'completed', updated_at = now()
		WHERE id = $1 AND status = 'pending'
	`, transactionId)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if update.RowsAffected() == 1 {
		creditRows, err := r.reconcileBalance(ctx, tx, accountId)
		if err != nil {
			return nil, fmt.Errorf("failed to credit account %s: %w", accountId, err)
		}
		if creditRows != 1 {
			return nil, fmt.Errorf("account %s not found", accountId)
		}
	}

	var transaction models.TransactionModel
	if err := tx.QueryRow(ctx, `
		SELECT id, from_account, to_account, amount, currency, status, created_at, updated_at
		FROM transactions WHERE id = $1
	`, transactionId).Scan(
		&transaction.ID,
		&transaction.FromAccount,
		&transaction.ToAccount,
		&transaction.Amount,
		&transaction.Currency,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to read transaction %s: %w", transactionId, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit deposit for account %s: %w", accountId, err)
	}

	return &transaction, nil
}

func (r *WalletRepository) ListTransactions(ctx context.Context, accountId string, offset, limit int) ([]models.TransactionModel, error) {
	query := `
        SELECT id, from_account, to_account, amount, currency, status, created_at, updated_at
        FROM transactions
        WHERE from_account = $1 OR to_account = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := r.db.Query(ctx, query, accountId, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions for account %s: %w", accountId, err)
	}
	defer rows.Close()

	var transactions []models.TransactionModel

	for rows.Next() {
		var transaction models.TransactionModel
		if err := rows.Scan(
			&transaction.ID,
			&transaction.FromAccount,
			&transaction.ToAccount,
			&transaction.Amount,
			&transaction.Currency,
			&transaction.Status,
			&transaction.CreatedAt,
			&transaction.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("Failed to scan transaction for account %s: %w", accountId, err)
		}
		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Failed to list transactions for account %s: %w", accountId, err)
	}

	return transactions, nil
}

func (r *WalletRepository) UpdateTransactionStatus(ctx context.Context, transactionId, userId, accountId string, status models.TransactionStatus) error {
	err := r.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		cmd, err := tx.Exec(ctx, `
			UPDATE transactions SET status = $1, updated_at = now() WHERE id = $2
		`, string(status), transactionId)
		if err != nil {
			return fmt.Errorf("Failed to update transaction status for id %s: %w", transactionId, err)
		}
		if cmd.RowsAffected() != 1 {
			return fmt.Errorf("No transaction found for id %s", transactionId)
		}

		return nil
	})
	if err != nil {
		r.Logger.LogWarn(ctx, "Failed to update transaction status for id %s: %v", transactionId, err)
		return fmt.Errorf("Failed to update transaction status for id %s: %w", transactionId, err)
	}

	// Releasing the idempotency enabling user account to make another transfer
	if err = r.ch.TransactionIdempotencyRelease(r.idempotencyKey(userId, accountId)); err != nil {
		r.Logger.LogWarn(ctx, "Failed to release idempotency for user %s account %s: %v", userId, accountId, err)
	}

	return nil
}

func (r *WalletRepository) WithdrawFunds(ctx context.Context, transactionId, accountId string, amount int64) (*models.TransactionModel, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // In case any error occurs the tx is rolled back completely ensuring durability

	update, err := tx.Exec(ctx, `
		UPDATE transactions SET status = 'completed', updated_at = now()
		WHERE id = $1 AND status = 'pending'
	`, transactionId)
	if err != nil {
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	if update.RowsAffected() == 1 {
		debitRows, err := r.reconcileBalance(ctx, tx, accountId)
		if err != nil {
			return nil, fmt.Errorf("Failed to debit account %s: %w", accountId, err)
		}
		if debitRows != 1 {
			return nil, fmt.Errorf("Account %s has insufficient balance", accountId)
		}
	}

	var transaction models.TransactionModel
	if err := tx.QueryRow(ctx, `
		SELECT id, from_account, to_account, amount, currency, status, created_at, updated_at
		FROM transactions WHERE id = $1
	`, transactionId).Scan(
		&transaction.ID,
		&transaction.FromAccount,
		&transaction.ToAccount,
		&transaction.Amount,
		&transaction.Currency,
		&transaction.Status,
		&transaction.CreatedAt,
		&transaction.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("Failed to read transaction %s: %w", transactionId, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("Failed to commit withdrawal for account %s: %w", accountId, err)
	}

	return &transaction, nil
}

func (r *WalletRepository) idempotencyKey(userId, accountId string) string {
	return userId + "_" + accountId
}

// Outbox operations
func (r *WalletRepository) OutboxInsert(ctx context.Context, outboxInsertConfig *domain.OutboxInsertConfig) error {

	// Destructuring struct
	aggregateType, aggregateId, eventType,
		payload, topic, partition_key, userId :=
		outboxInsertConfig.AggregateType, outboxInsertConfig.AggregateId,
		outboxInsertConfig.EventType, outboxInsertConfig.Payload,
		outboxInsertConfig.Topic, outboxInsertConfig.Partition_key, outboxInsertConfig.UserId

	// Performing idempotency check in here before writing into the outbox table.
	// partition_key is the account id, so this locks the (user, account) pair —
	// only one transaction may be in flight for that account at a time.
	//
	// Also as per design decision we prevent failures in case cache operations throws an error
	// This ensures invalid redis state won't disrupt the transaction processing
	// And subsequent step will ensure idempotency themselves
	idempotencyKey := r.idempotencyKey(userId, partition_key)

	// Cached idempotency check not only checks for the idempotency but inserts it in case it's not present,
	// meaning it's a pass for the outbox insert
	ok, err := r.ch.TransactionIdempotencyCheck(idempotencyKey)

	if err != nil {
		r.LogWarn(ctx, "%s Idempotency cache operation failed for: %s, with error:%v", aggregateType, idempotencyKey, err)

		// In case Cache idempotency lookup fails we have a fallback that will ensure the idempotency through persistence
		inProgress, pgErr := r.outboxIdempotencyCheck(ctx, idempotencyKey)
		if pgErr != nil {
			r.LogWarn(ctx, "%s Persistence idempotency check failed: %s, with error:%v", aggregateType, idempotencyKey, pgErr)
		} else if inProgress {
			r.LogWarn(ctx, "%s Idempotency check (postgres fallback) failed, account %s already has a transaction in progress", aggregateType, partition_key)
			return fmt.Errorf("A transaction is already in progress for operator %s", partition_key)
		}
	} else if !ok {
		r.LogWarn(ctx, "%s Idempotency check failed, account %s already has a transaction in progress", aggregateType, partition_key)
		return fmt.Errorf("Transaction is already in progress for operator %s", partition_key)
	}

	r.Logger.LogInfo(ctx, "outbox insert: aggregate_type=%s aggregate_id=%s event_type=%s topic=%s partition_key=%s", aggregateType, aggregateId, eventType, topic, partition_key)

	query := `
		INSERT INTO transaction_outbox (aggregate_type, aggregate_id, event_type, topic, partition_key, payload, idempotency_key) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, aggregate_type, aggregate_id, event_type, topic, partition_key, status, error_message, created_at, published_at, updated_at, payload, idempotency_key
	`

	var outboxRow models.OutboxModel

	err = r.db.Pool.QueryRow(ctx, query, aggregateType, aggregateId, eventType, topic, partition_key, payload, idempotencyKey).Scan(
		&outboxRow.ID,
		&outboxRow.AggregateType,
		&outboxRow.AggregateId,
		&outboxRow.EventType,
		&outboxRow.Topic,
		&outboxRow.PartitionKey,
		&outboxRow.Status,
		&outboxRow.ErrorMessage,
		&outboxRow.CreatedAt,
		&outboxRow.PublishedAt,
		&outboxRow.UpdatedAt,
		&outboxRow.Payload,
		&outboxRow.IdempotencyKey,
	)

	if err != nil {
		r.Logger.LogWarn(ctx, "Failed to insert outbox with aggregateType: %s, aggregateId:%s", aggregateType, aggregateId)
		// No message will ever reach the consumer to release this lock — release it now.
		if relErr := r.ch.TransactionIdempotencyRelease(idempotencyKey); relErr != nil {
			r.Logger.LogWarn(ctx, "Failed to release idempotency lock %s after failed outbox insert: %v", idempotencyKey, relErr)
		}
		return fmt.Errorf("Failed to insert outbox with aggregateType:%s, aggregateId:%s in DB: %w", aggregateType, aggregateId, err)
	}

	return nil
}

func (r *WalletRepository) IdempotencyRelease(ctx context.Context, transactionId, userId, accountId string) error {
	if err := r.ch.TransactionIdempotencyRelease(r.idempotencyKey(userId, accountId)); err != nil {
		r.Logger.LogWarn(ctx, "Failed to release idempotency for transaction %s, user %s account %s: %v", transactionId, userId, accountId, err)
		return fmt.Errorf("failed to release idempotency for transaction %s: %w", transactionId, err)
	}

	return nil
}

func (r *WalletRepository) outboxIdempotencyCheck(ctx context.Context, idempotencyKey string) (bool, error) {
	var inProgress bool

	err := r.db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM transaction_outbox o
			JOIN transactions t ON t.id = o.aggregate_id
			WHERE o.idempotency_key = $1
			  AND t.status = 'pending'
		)
	`, idempotencyKey).Scan(&inProgress)
	if err != nil {
		return false, fmt.Errorf("failed to check postgres idempotency for lock %s: %w", idempotencyKey, err)
	}

	return inProgress, nil
}

func (r *WalletRepository) OutboxUpdate(ctx context.Context, aggregateId, status string) error {
	cmd, err := r.db.Pool.Exec(ctx, `
		UPDATE transaction_outbox
		SET status = $1, updated_at = now(), published_at = CASE WHEN $1 = 'published'::outbox_status THEN now() ELSE published_at END WHERE aggregate_id = $2
	`, status, aggregateId)
	if err != nil {
		return fmt.Errorf("failed to update outbox status for aggregate_id %s: %w", aggregateId, err)
	}
	if cmd.RowsAffected() != 1 {
		return fmt.Errorf("no outbox row found for aggregate_id %s", aggregateId)
	}

	return nil
}

func (r *WalletRepository) OutboxMarkPublishFailure(ctx context.Context, aggregateId, errMsg string) (string, error) {
	// Dynamically re-assigned via env variables for the ability to control the recovery effort
	var outboxBackoffMaxSeconds = env.GetInt("OUTBOX_BACKOFF_SECONDS", 60)
	var outboxMaxPublishRetries = env.GetInt("OUTBOX_MAX_REPUBLISH_RETRIES", 4)

	var status string
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE transaction_outbox
		SET retry_count = retry_count + 1,
			error_message = $2,
			status = CASE WHEN retry_count + 1 >= $3 THEN 'failed'::outbox_status ELSE 'pending'::outbox_status END,
			next_attempt_at = now() + (LEAST(1 << (retry_count + 1), $4) * interval '1 second'),
			updated_at = now()
		WHERE aggregate_id = $1
		RETURNING status
	`, aggregateId, errMsg, outboxMaxPublishRetries, outboxBackoffMaxSeconds).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("failed to record publish failure for aggregate_id %s: %w", aggregateId, err)
	}

	return status, nil
}

func (r *WalletRepository) OutboxDelete(ctx context.Context, aggregateId string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM transaction_outbox WHERE aggregate_id = $1`, aggregateId)
	if err != nil {
		return fmt.Errorf("failed to delete outbox row for aggregate_id %s: %w", aggregateId, err)
	}

	return nil
}

func (r *WalletRepository) OutboxGet(ctx context.Context, aggregateId string) (*models.OutboxModel, error) {
	var outboxRow models.OutboxModel

	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, topic, partition_key, status, error_message, created_at, published_at, updated_at, payload
		FROM transaction_outbox
		WHERE aggregate_id = $1
	`, aggregateId).Scan(
		&outboxRow.ID,
		&outboxRow.AggregateType,
		&outboxRow.AggregateId,
		&outboxRow.EventType,
		&outboxRow.Topic,
		&outboxRow.PartitionKey,
		&outboxRow.Status,
		&outboxRow.ErrorMessage,
		&outboxRow.CreatedAt,
		&outboxRow.PublishedAt,
		&outboxRow.UpdatedAt,
		&outboxRow.Payload,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get outbox row for aggregate_id %s: %w", aggregateId, err)
	}

	return &outboxRow, nil
}

// OutboxPendingGetBatch owns the full transaction lifecycle internally.
// FOR UPDATE SKIP LOCKED only needs to hold the row locks for the duration of
// this read — it commits - updates the status to claimed before returning, so concurrent callers never pick
// up the same row twice, but never see a transaction handle to manage.
func (r *WalletRepository) OutboxPendingGetBatch(ctx context.Context, limit int) ([]*models.OutboxModel, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Reclaiming operation on stale claimed rows in case of system being down, preventing messages from being lost since
	// Moving the status back to pending for retried claiming and processing.
	if _, err := tx.Exec(ctx, `
		UPDATE transaction_outbox
		SET status = 'pending', updated_at = now()
		WHERE status = 'claimed' AND updated_at < now() - interval '30 seconds'
	`); err != nil {
		return nil, fmt.Errorf("failed to reclaim stale outbox claims: %w", err)
	}

	rows, err := tx.Query(ctx, `
		WITH claimed AS (
			SELECT id
			FROM transaction_outbox
			WHERE status = 'pending' AND next_attempt_at <= now()
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE transaction_outbox
		SET status = 'claimed', updated_at = now()
		FROM claimed
		WHERE transaction_outbox.id = claimed.id
		RETURNING transaction_outbox.id, transaction_outbox.aggregate_type, transaction_outbox.aggregate_id, transaction_outbox.event_type, transaction_outbox.topic, transaction_outbox.partition_key, transaction_outbox.payload, transaction_outbox.created_at, transaction_outbox.idempotency_key
	`, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to query outbox batch: %w", err)
	}

	defer rows.Close()

	// Outbox slice to store the pending rows at
	var outboxRows []*models.OutboxModel

	for rows.Next() {
		var outboxRow models.OutboxModel
		err := rows.Scan(&outboxRow.ID, &outboxRow.AggregateType, &outboxRow.AggregateId, &outboxRow.EventType, &outboxRow.Topic, &outboxRow.PartitionKey, &outboxRow.Payload, &outboxRow.CreatedAt, &outboxRow.IdempotencyKey)

		if err != nil {
			return nil, fmt.Errorf("failed to retrieve outbox row: %w", err)
		}
		outboxRows = append(outboxRows, &outboxRow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read outbox batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit outbox batch claim: %w", err)
	}

	return outboxRows, nil
}

// Outbox access
