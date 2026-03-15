package models

type Expenses struct {
	expenseID int
	userID    int
	name      string
	category  string
	amount    float64
	createAt  string
}
