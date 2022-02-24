package flags

// Response - Servers send a response containing all the data output from their command
// implementations, as well as basic things for tracking back to the correct clients.
type Response struct {
	ID        string
	Stdout    []byte // Here goes the output of user-defined command implementation
	RequestID string // The hash of the requesting command data
	Async     bool   // If true, the ID is then a task ID
	err       error  // There is a method for querying this
}
