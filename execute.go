package flags

import (
	"encoding/json"
	"errors"
	"fmt"
)

// executePreRuns executes all the pre-run functions for all parent
// commands (if persistent) and for the currently executed one.
func (p *Client) executePreRuns(s *parseState) ([]string, error) {
	// Execute all persistent pre-runs for
	// parent commands including the parser itself.
	for _, name := range s.lookup.commandList {
		parent := s.lookup.commands[name]
		if parent == nil || parent.PersistentPreRun == nil {
			continue
		}

		cmd, _ := parent.data.(Commander)
		perr := parent.PersistentPreRun(cmd, s.retargs) // Change retargs
		if perr != nil {
			return s.retargs, perr
		}
	}

	// Run the command's pre-runs if any
	if s.command.PersistentPreRun != nil {
		cmd := s.command.data.(Commander)
		if perr := s.command.PersistentPreRun(cmd, s.retargs); perr != nil {
			return s.retargs, perr
		}
	}
	if s.command.PreRun != nil {
		cmd := s.command.data.(Commander)
		if perr := s.command.PreRun(cmd, s.retargs); perr != nil {
			return s.retargs, perr
		}
	}

	return s.retargs, nil
}

// executeMain is in charge of appropriately executing the current command,
// according to the client's setup, the implementations of the command type, etc.
func (p *Client) executeMain(s *parseState) (err error) {
	// Assess the command's remote/local implementations
	command, remote := s.command.data.(CommanderClient)

	// Local Execution Steps (common) -----------------------------------

	// If the command is a purely local one, execute it and return.
	if !remote {
		cmd := s.command.data.(Commander)
		return cmd.Execute(s.retargs)
	}

	// Else, we are executing a remote command.
	// First, execute the pre-send function implementation.
	if err = command.Execute(s.retargs); err != nil {
		return err
	}

	// If the client has no Request sending handler, we won't
	// ever receive a response: usually it's because the user
	// has manually invoked the handler in the Execute() function,
	// so, we don't have to trigger Response() ourselves.
	if p.clientHandler == nil && p.asyncHandler == nil {
		return
	}

	//  Remote Execution Steps ------------------------------------------

	// The request to be sent to the remote flag server
	req, err := p.makeRequest(s)
	if err != nil {
		return
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request to JSON: %s", err)
	}

	// If the current call is asynchronous (eg. the response will be
	// received later), use the appropriate handler and return
	if req.Async {
		if p.asyncHandler == nil {
			return errors.New("request is marked asynchronous, but client has no async handler")
		}
		p.mutex.RLock()
		p.pendingReqs[req.ID] = req
		p.pendingCmds[req.ID] = command
		p.mutex.RUnlock()
		return p.asyncHandler(reqData)
	}

	// Else immediately call the remote and execute the response
	respData, err := p.clientHandler(reqData)
	if err != nil {
		return fmt.Errorf("request failed: %s", err)
	}

	// Response Execution Steps -----------------------------------------

	resp := Response{}
	if err = json.Unmarshal(respData, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response from bytes: %s", err)
	}

	// Execute the post-receive implementation
	return command.Response(resp.Stdout, resp.err)
}

// executePostRuns is identical to executePreRuns, for post-execution runs.
func (p *Client) executePostRuns(s *parseState) ([]string, error) {
	// Execute all persistent post-runs for
	// parent commands including the parser itself.
	for _, name := range s.lookup.commandList {
		parent := s.lookup.commands[name]
		if parent == nil || parent.PersistentPostRun == nil {
			continue
		}

		cmd, _ := parent.data.(Commander)
		perr := parent.PersistentPostRun(cmd, s.retargs) // Change retargs
		if perr != nil {
			return s.retargs, perr
		}
	}

	// Run the command's post-runs if any
	if s.command.PersistentPostRun != nil {
		cmd := s.command.data.(Commander)
		if perr := s.command.PersistentPostRun(cmd, s.retargs); perr != nil {
			return s.retargs, perr
		}
	}
	if s.command.PostRun != nil {
		cmd := s.command.data.(Commander)
		if perr := s.command.PostRun(cmd, s.retargs); perr != nil {
			return s.retargs, perr
		}
	}

	return s.retargs, nil
}
