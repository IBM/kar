package runtime

// The contents of this file really should be in cmd/kar/kar-api-docs.go.
// These structs are defined here instead to work around a limitation of the
// swagger tool -- it doesn't support reponses whose bodues are arrays
// whose element type is imported from a different package.

// swagger:response response200SubscriptionGetAllResult
type response200SubscriptionGetAllResult struct {
	// An array containing all matching subscriptions
	Body []Source
}

// swagger:response response200ReminderGetAllResult
type response200ReminderGetAllResult struct {
	// An array containing all matching reminders
	// Example: [{ Actor: { Type: 'Foo', ID: '22' }, id: 'ticker', path: '/echo', targetTime: '2020-04-14T14:17:51.073Z', period: 5000000000, encodedData: '{"msg":"hello"}' }, { Actor: { Type: 'Foo', ID: '22' }, id: 'once', path: '/echo', targetTime: '2020-04-14T14:20:00Z', encodedData: '{"msg":"carpe diem"}' }]
	Body []Reminder
}
