// Package worker drives the background polling of the external accrual
// service. It claims batches of pending orders from storage, queries the
// accrual API and applies the results to the user's balance.
package worker
