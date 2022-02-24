package flags

// Request - Clients commands transmit their requests to their server-side equivalent
// with this type. It provides basic functionality for tracking which client has sent
// a request to which server, because several command servers might be reachable.
type Request struct {
	ID       string // Generally the hash of the command data
	Type     string // The name of command Go type
	Async    bool   // Does this request works through async tasking ?
	ClientID string // The ID of the client instance who made the request
	TargetID string // The ID of the target command server

	Args     []string               // Any remaining args from the parsed command
	Data     []byte                 // Here goes your user-defined types, marshalled
	Options  map[string]interface{} // All options set by the command line
	Groups   map[string][]byte      //
	Commands map[string][]byte      //
}
