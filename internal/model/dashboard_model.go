package model

import "time"

type DashboardStatsResponse struct {
	TotalItems int64 `json:"total_items"`
	StockIn    int64 `json:"stock_in"`
	StockOut   int64 `json:"stock_out"`

	TotalBudget int64 `json:"total_budged"`
	BudgetIn    int64 `json:"budget_in"`
	BudgetOut   int64 `json:"budget_out"`

	MonthlyBudget []MonthlyBudgetStat  `json:"monthly_budget"`
	ExpenseByType []ExpenseComposition `json:"expense_composition"`

	SystemActivities []SystemActivity `json:"system_activities"`
}

type MonthlyBudgetStat struct {
	Month string `json:"month"`
	In    int64  `json:"in"`
	Out   int64  `json:"out"`
}

type ExpenseComposition struct {
	Category string `json:"category"`
	Amount   int64  `json:"amount"`
}

type SystemActivity struct {
	ID          int       `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
