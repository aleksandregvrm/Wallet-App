package domain

import "time"

// Domain User is shared across both services. That is why we moved it to shared/domain directory
type User struct {
	ID        string
	Name      string
	Username  string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Domain Transaction Shared across all the services, There is no service specific version available for the domain transaction
type Transaction struct {
	ID          string
	FromAccount string
	ToAccount   string
	Amount      int64
	Currency    string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
