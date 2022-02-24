package flags

import (
	"context"
	"sync"
)

// CommanderClient is the only interface that any concrete type must implement
// in order to be able to send its command flag set to a remote command peer.
type CommanderClient interface {
	// Discover should return the fully qualified (pkg/path.Type) name
	// of the command struct used by BOTH the client and the remote
	// instance. This is absolutely primordial, since there is no
	// other way for commands to be discovered and executed remotely.
	// Might be removed when it will be possible to access this with
	// the stdlib reflect package, which is not possible at the moment.
	Discover() interface{}

	// Execute is the pre-send implementation of a client-side command.
	// Its body can be empty, but its return value should involve a call
	// to a reflag.Client, like myclient.Request(myCommand, args).
	// The error return should thus be either one defined in your
	// implementation or one throwned by the reflag client, either
	// when sending or when receiving the response.
	Commander

	// Response is the post-receive implementation of your command, and
	// most of the time automatically triggered by the reflag.Client.
	//
	// Binding:
	// Note that if you bind a command that implements this method, your
	// command will be considered remote by default. If you want to prevent
	// this, you must either:
	// - Redefine the type and declare only an Execute() method on it,
	//   which does not prevent you from accessing command flags and state.
	// - Bind the command with AddLocal() instead of AddCommand()
	//
	// Parameters:
	// @out     is the stdout buffered result of the remote command,
	//          (the `out []byte` return parameter of its Execute() func)
	// @err     is the stderr buffered result of the remote command,
	//          wrapped into an error type for more conventional use.
	Response(out []byte, err error) error
}

// Client provides command-line cmd/arg/opt parsing. It can contain
// several command/option groups, each with their own sets of subcommands/options,
// with any level of nesting, and with many customizations for each component.
type Client struct {
	// CLI (Local)
	*Command                      // Embedded, see Command for more information
	Usage                 string  // A usage string to be displayed in the help message.
	Options               Options // Option flags changing the behavior of the parser.
	EnvNamespaceDelimiter string  // EnvNamespaceDelimiter separates group env namespaces and env keys
	internalError         error

	// Request-time (Remote)
	id            string                 // The ID of the client be overridden by a call to SetID()
	asyncHandler  AsyncHandler           // Sends requests asynchronously, letting the library doing cmd.Response()
	asyncSetter   func() bool            // A function called each call, to determine if run it async
	asyncOverride bool                   // Does the reflag client still executes the command after callback ?
	mustEncrypt   bool                   // This is so that there can't be any ambiguity client-side
	seeder        func() int64           // A function giving the correct encryption seed at each call.
	clientHandler ClientHandler          // The function used to actually send the requests
	contextMaker  func() context.Context // A context that can be set by users

	// Response-time (Remote)
	pendingReqs map[string]Request         // All requests waiting for their responses
	pendingCmds map[string]CommanderClient // One command for one Request

	// UnknownOptionsHandler is a function which gets called when the parser
	// encounters an unknown option. The function receives the unknown option
	// name, a SplitArgument which specifies its value if set with an argument
	// separator, and the remaining command line arguments.
	// It should return a new list of remaining arguments to continue parsing,
	// or an error to indicate a parse failure.
	UnknownOptionHandler func(option string, arg SplitArgument, args []string) ([]string, error)

	// Others
	mutex *sync.RWMutex
}

// NewClient - Creates a new reflag command Client.
func NewClient(name string, opts Options) *Client {
	client := &Client{
		// Local
		Command:               newCommand(name, "", "", nil),
		EnvNamespaceDelimiter: "_",
		Options:               opts,

		// Remote
		pendingReqs: map[string]Request{},
		// pendingCmds: map[string]Command{},

		// Others
		mutex: &sync.RWMutex{},
	}

	client.Command.parent = client

	return client
}

// SetID - Overrride the automatically generated ID for this client.
// This is useful when your client application already makes use of one.
func (c *Client) SetID(id string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.id = id
}

// SeedEncrypt - When your command servers are encrypted, the client must know how
// to modify the keys (fields) of your user-defined types so that the server can
// correctly unmarshal them. The seed is need to recompute encryption.
// In addition, because each server build might have different encryption seed,
// you must therefore pass a function that will determine the correct key seed.
func (c *Client) SeedEncrypt(seeder func() int64) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.seeder = seeder
	c.mustEncrypt = true
}

// ContextGenerate is a function that is used for generating call contexts, which
// are used either when calling external RPC callers (eg. gRPC), or internally.
func (c *Client) ContextGenerate(maker func() context.Context) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.contextMaker = maker
}

//
// Transports/Handlers Setup ------------------------------------------
//

// ClientHandler is a very generic function that takes arbitrary
// byte data, sends it to some destination, and then returns a response.
// The function is willingly generic, to let you free of the underlying
// network/transport/RPC stack you want to use to forward cmd requests.
type ClientHandler func(req []byte) (resp []byte, err error)

// Request is used to specify the handler used to actually send requests,
// which can be anything: over the network or not, various things can be used.
func (c *Client) Request(handler ClientHandler) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.clientHandler = handler
}

// AsyncHandler is the asynchronous equivalent of ClientHandler, with one
// key difference: its return parameters do not include a `data []byte` argument
// containing the remote response's content, since it has not yet been executed.
// The error returned is thus only one related to sending the request itself.
//
// When using this handler (eg. binding it to your flags client), you will need
// to make use of client.AsyncResponse() function to manually pass the response
// once you have it.
type AsyncHandler func(req []byte) error

// makeRequest populates a request to be sent to the remote server,
// including all options that might be contained in the command's
// parent groups (marked persistent).
func (p *Client) makeRequest(s *parseState) (req Request, err error) {
	req.Async = p.asyncSetter()

	return
}
