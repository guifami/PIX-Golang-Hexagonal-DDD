package usecase_test

import (
	"context"
	"errors"

	"go-api/internal/domain/account"
	"go-api/internal/domain/pixkey"
	"go-api/internal/domain/transaction"
)

// ── AccountRepository mock ────────────────────────────────────────────────

type mockAccountRepo struct {
	accounts        map[string]*account.Account
	byCPF           map[string]*account.Account
	createErr       error
	findErr         error
	updateErr       error
	updateErrForID  string // if set, Update fails only for this account ID
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{
		accounts: make(map[string]*account.Account),
		byCPF:    make(map[string]*account.Account),
	}
}

func (m *mockAccountRepo) Create(_ context.Context, acc *account.Account) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.accounts[acc.ID] = acc
	m.byCPF[acc.CPF.String()] = acc
	return nil
}

func (m *mockAccountRepo) FindByID(_ context.Context, id string) (*account.Account, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	a, ok := m.accounts[id]
	if !ok {
		return nil, account.ErrAccountNotFound
	}
	return a, nil
}

func (m *mockAccountRepo) FindByCPF(_ context.Context, cpf string) (*account.Account, error) {
	if a, ok := m.byCPF[cpf]; ok {
		return a, nil
	}
	return nil, account.ErrAccountNotFound
}

func (m *mockAccountRepo) Update(_ context.Context, acc *account.Account) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.updateErrForID != "" && acc.ID == m.updateErrForID {
		return errDB
	}
	m.accounts[acc.ID] = acc
	return nil
}

// ── PixKeyRepository mock ─────────────────────────────────────────────────

type mockPixKeyRepo struct {
	keys       map[string]*pixkey.PixKey
	count      int
	countErr   error
	createErr  error
	findErr    error
	deleteErr  error
	listErr    error
}

func newMockPixKeyRepo() *mockPixKeyRepo {
	return &mockPixKeyRepo{keys: make(map[string]*pixkey.PixKey)}
}

func (m *mockPixKeyRepo) Create(_ context.Context, k *pixkey.PixKey) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.keys[k.Value] = k
	return nil
}

func (m *mockPixKeyRepo) FindByValue(_ context.Context, value string) (*pixkey.PixKey, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	k, ok := m.keys[value]
	if !ok {
		return nil, pixkey.ErrKeyNotFound
	}
	return k, nil
}

func (m *mockPixKeyRepo) FindByAccountID(_ context.Context, accountID string) ([]pixkey.PixKey, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []pixkey.PixKey
	for _, k := range m.keys {
		if k.AccountID == accountID {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *mockPixKeyRepo) CountByAccountID(_ context.Context, _ string) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func (m *mockPixKeyRepo) Delete(_ context.Context, value string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.keys, value)
	return nil
}

// ── TransactionRepository mock ────────────────────────────────────────────

type mockTxRepo struct {
	txs                       map[string]*transaction.PixTransaction
	createErr                 error
	findErr                   error
	updateErr                 error
	updateErrAfter            int // updateErr is returned after this many successful Update calls
	updateCalls               int
	listErr                   error
	resetStatusAfterProcessing bool // if true, corrupts tx.Status to Pending after PROCESSING update
}

func newMockTxRepo() *mockTxRepo {
	return &mockTxRepo{txs: make(map[string]*transaction.PixTransaction), updateErrAfter: -1}
}

func (m *mockTxRepo) Create(_ context.Context, tx *transaction.PixTransaction) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.txs[tx.ID] = tx
	return nil
}

func (m *mockTxRepo) FindByID(_ context.Context, id string) (*transaction.PixTransaction, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	tx, ok := m.txs[id]
	if !ok {
		return nil, transaction.ErrTransactionNotFound
	}
	return tx, nil
}

func (m *mockTxRepo) FindByPayerAccountID(_ context.Context, accountID string) ([]transaction.PixTransaction, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []transaction.PixTransaction
	for _, tx := range m.txs {
		if tx.PayerAccountID == accountID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

func (m *mockTxRepo) Update(_ context.Context, tx *transaction.PixTransaction) error {
	if m.updateErr != nil && m.updateCalls >= m.updateErrAfter {
		return m.updateErr
	}
	m.updateCalls++
	if m.resetStatusAfterProcessing && tx.Status == transaction.Processing {
		tx.Status = transaction.Pending
	}
	m.txs[tx.ID] = tx
	return nil
}

// ── EventPublisher mock ───────────────────────────────────────────────────

type mockPublisher struct {
	publishedInitiated  []*transaction.PixTransaction
	publishedCompleted  []*transaction.PixTransaction
	publishedFailed     []*transaction.PixTransaction
	initiatedErr        error
	completedErr        error
	failedErr           error
}

func (m *mockPublisher) PublishPaymentInitiated(_ context.Context, tx *transaction.PixTransaction) error {
	if m.initiatedErr != nil {
		return m.initiatedErr
	}
	m.publishedInitiated = append(m.publishedInitiated, tx)
	return nil
}

func (m *mockPublisher) PublishPaymentCompleted(_ context.Context, tx *transaction.PixTransaction) error {
	if m.completedErr != nil {
		return m.completedErr
	}
	m.publishedCompleted = append(m.publishedCompleted, tx)
	return nil
}

func (m *mockPublisher) PublishPaymentFailed(_ context.Context, tx *transaction.PixTransaction) error {
	if m.failedErr != nil {
		return m.failedErr
	}
	m.publishedFailed = append(m.publishedFailed, tx)
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

var errDB = errors.New("database error")

func makeAccount(id, cpf string, balanceCents int64) *account.Account {
	a, _ := account.NewAccount("Test User", cpf)
	a.ID = id
	a.Balance = account.Money(balanceCents)
	return a
}
