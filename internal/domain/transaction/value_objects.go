package transaction

type TransactionStatus string

const (
	Pending    TransactionStatus = "PENDING"
	Processing TransactionStatus = "PROCESSING"
	Completed  TransactionStatus = "COMPLETED"
	Failed     TransactionStatus = "FAILED"
)
