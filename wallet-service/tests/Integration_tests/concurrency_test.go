package integration_tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-task-wallet-service/wallet-service/internal/domain"
	"go-task-wallet-service/wallet-service/internal/repository"
	"go-task-wallet-service/wallet-service/tests/container"

	"github.com/google/uuid"
)

// Test for concurrent transactions which should not deadlock due to inconsistent lock ordering. Two goroutines attempt to transfer funds in opposite directions between two accounts. The test ensures that all transfers complete without deadlock and that the total balance across both accounts remains constant.
func TestWalletRepository_TransferFunds_ConcurrentOpposingTransfers_NoDeadlock(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	accountA := insertAccount(t, env, repo, "USD")
	accountB := insertAccount(t, env, repo, "USD")

	const startingBalance = 100000
	for _, acc := range []string{accountA.ID, accountB.ID} {
		depositId := uuid.NewString()
		if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", acc, acc, "USD", startingBalance); err != nil {
			t.Fatalf("setup: failed to create pending deposit for %s: %v", acc, err)
		}
		if _, err := repo.DepositFunds(ctx, depositId, acc, "USD", startingBalance); err != nil {
			t.Fatalf("setup: failed to fund account %s: %v", acc, err)
		}
	}

	const attemptsPerDirection = 15
	const transferAmount = 100
	var wg sync.WaitGroup

	runDirection := func(from, to string) {
		defer wg.Done()
		for i := 0; i < attemptsPerDirection; i++ {
			transferId := uuid.NewString()
			if err := repo.CreatePendingTransaction(ctx, transferId, "transfer", from, to, "USD", transferAmount); err != nil {
				t.Errorf("failed to create pending transfer %s->%s: %v", from, to, err)
				return
			}
			if _, err := repo.TransferFunds(ctx, transferId, from, to, "USD", transferAmount); err != nil {
				t.Errorf("unexpected transfer error %s->%s: %v", from, to, err)
				return
			}
		}
	}

	wg.Add(2)
	go runDirection(accountA.ID, accountB.ID)
	go runDirection(accountB.ID, accountA.ID)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatalf("timed out waiting for opposing transfers to complete — possible deadlock from inconsistent lock ordering")
	}

	fromFound, err := repo.FindAccountById(ctx, accountA.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	toFound, err := repo.FindAccountById(ctx, accountB.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every successful transfer moves money between A and B without creating
	// or destroying any — the total across both accounts must be conserved
	// regardless of how the individual transfers interleaved.
	total := fromFound.Balance + toFound.Balance
	if total != 2*startingBalance {
		t.Fatalf("expected total balance to be conserved at %d, got: %d (A=%d B=%d)", 2*startingBalance, total, fromFound.Balance, toFound.Balance)
	}
}

// Exercises reconcileBalance's "computed.balance >= 0" guard under concurrent
// withdrawals from the same account — more withdrawals are attempted than the
// account can cover, so some must fail, but the account must never go negative.
func TestWalletRepository_WithdrawFunds_ConcurrentWithdrawals_NeverOverdraws(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	account := insertAccount(t, env, repo, "USD")

	const startingBalance = 1000
	const withdrawAmount = 100
	const attempts = 20 // requests up to 2000 against a balance of 1000

	depositId := uuid.NewString()
	if err := repo.CreatePendingTransaction(ctx, depositId, "deposit", account.ID, account.ID, "USD", startingBalance); err != nil {
		t.Fatalf("setup: failed to create pending deposit: %v", err)
	}
	if _, err := repo.DepositFunds(ctx, depositId, account.ID, "USD", startingBalance); err != nil {
		t.Fatalf("setup: failed to fund account: %v", err)
	}

	var wg sync.WaitGroup
	var succeeded int64
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			transactionId := uuid.NewString()
			if err := repo.CreatePendingTransaction(ctx, transactionId, "withdrawal", account.ID, account.ID, "USD", withdrawAmount); err != nil {
				t.Errorf("failed to create pending withdrawal: %v", err)
				return
			}
			if _, err := repo.WithdrawFunds(ctx, transactionId, account.ID, withdrawAmount); err == nil {
				atomic.AddInt64(&succeeded, 1)
			}
		}()
	}
	wg.Wait()

	t.Logf("concurrent withdrawals: succeeded=%d out of %d attempts", succeeded, attempts)

	found, err := repo.FindAccountById(ctx, account.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Balance < 0 {
		t.Fatalf("account balance went negative: %d", found.Balance)
	}
	expectedBalance := int64(startingBalance) - succeeded*withdrawAmount
	if found.Balance != expectedBalance {
		t.Fatalf("expected balance %d (started at %d, %d withdrawals succeeded), got: %d", expectedBalance, startingBalance, succeeded, found.Balance)
	}
}

// Exercises OutboxPendingGetBatch's "FOR UPDATE SKIP LOCKED" batch read under
// concurrent pollers (simulating multiple wallet-service replicas). Each
// poller immediately marks what it fetches as published, mirroring the real
// relay loop, so the invariant under test is: every row is delivered to
// exactly one poller, exactly once.
func TestWalletRepository_OutboxPendingGetBatch_ConcurrentPollers_NoDoubleDelivery(t *testing.T) {
	env := container.SetupTestContainerEnv(t)
	repo := repository.NewWalletRepository(env.DB, env.Cache)
	ctx := context.Background()

	const totalRows = 20
	for i := 0; i < totalRows; i++ {
		id := uuid.NewString()
		err := repo.OutboxInsert(ctx, &domain.OutboxInsertConfig{
			AggregateType: "transaction", AggregateId: id, EventType: "transaction.deposit.requested",
			Payload: map[string]interface{}{"ID": id}, UserId: uuid.NewString(), Topic: "wallet.events.v1", Partition_key: uuid.NewString(),
		})
		if err != nil {
			t.Fatalf("setup: failed to insert outbox row %d: %v", i, err)
		}
	}

	var mu sync.Mutex
	seen := make(map[string]int)
	var delivered int64

	poll := func() {
		for atomic.LoadInt64(&delivered) < totalRows {
			batch, err := repo.OutboxPendingGetBatch(ctx, 5)
			if err != nil {
				t.Errorf("poller: failed to fetch batch: %v", err)
				return
			}
			if len(batch) == 0 {
				return
			}
			for _, row := range batch {
				mu.Lock()
				seen[row.AggregateId]++
				mu.Unlock()
				if err := repo.OutboxUpdate(ctx, row.AggregateId, "published"); err != nil {
					t.Errorf("poller: failed to mark %s published: %v", row.AggregateId, err)
				}
				atomic.AddInt64(&delivered, 1)
			}
		}
	}

	var wg sync.WaitGroup
	const pollers = 4
	wg.Add(pollers)
	for i := 0; i < pollers; i++ {
		go func() {
			defer wg.Done()
			poll()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatalf("timed out waiting for concurrent pollers to drain the outbox")
	}

	if len(seen) != totalRows {
		t.Fatalf("expected %d distinct rows delivered, got: %d", totalRows, len(seen))
	}
	for id, count := range seen {
		if count != 1 {
			t.Fatalf("row %s was delivered %d times, expected exactly once", id, count)
		}
	}
}
