// Package backend provides interfaces that the CLI uses to interact with
// Terraform. A backend provides the abstraction that allows the same CLI
// to simultaneously support both local and remote operations for seamlessly
// using Terraform in a team environment.
package backend

import (
	"context"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// Backend is the minimal interface that must be implemented to enable Terraform.
type Backend interface {
	// Ask for input and configure the backend. Similar to
	// terraform.ResourceProvider.
	Input(terraform.UIInput, *terraform.ResourceConfig) (*terraform.ResourceConfig, error)
	Validate(*terraform.ResourceConfig) ([]string, []error)
	Configure(*terraform.ResourceConfig) error

	// State returns the current state for this environment. This state may
	// not be loaded locally: the proper APIs should be called on state.State
	// to load the state.
	State() (state.State, error)
}

// Enhanced implements additional behavior on top of a normal backend.
//
// Enhanced backends allow customizing the behavior of Terraform operations.
// This allows Terraform to potentially run operations remotely, load
// configurations from external sources, etc.
type Enhanced interface {
	Backend

	// Operation performs a Terraform operation such as refresh, plan, apply.
	// It is up to the implementation to determine what "performing" means.
	// This DOES NOT BLOCK. The context returned as part of RunningOperation
	// should be used to block for completion.
	Operation(context.Context, *Operation) (*RunningOperation, error)
}

// Local implements additional behavior on a Backend that allows local
// operations in addition to remote operations.
//
// This enables more behaviors of Terraform that require more data such
// as `console`, `import`, `graph`. These require direct access to
// configurations, variables, and more. Not all backends may support this
// so we separate it out into its own optional interface.
type Local interface {
	// Context returns a runnable terraform Context. The operation parameter
	// doesn't need a Type set but it needs other options set such as Module.
	Context(*Operation) (*terraform.Context, state.State, error)
}

// An operation represents an operation for Terraform to execute.
//
// Note that not all fields are supported by all backends and can result
// in an error if set. All backend implementations should show user-friendly
// errors explaining any incorrectly set values. For example, the local
// backend doesn't support a PlanId being set.
//
// The operation options are purposely designed to have maximal compatibility
// between Terraform and Terraform Servers (a commercial product offered by
// HashiCorp). Therefore, it isn't expected that other implementation support
// every possible option. The struct here is generalized in order to allow
// even partial implementations to exist in the open, without walling off
// remote functionality 100% behind a commercial wall. Anyone can implement
// against this interface and have Terraform interact with it just as it
// would with HashiCorp-provided Terraform Servers.
type Operation struct {
	// Type is the operation to perform.
	Type OperationType

	// PlanId is an opaque value that backends can use to execute a specific
	// plan for an apply operation.
	//
	// PlanOutBackend is the backend to store with the plan. This is the
	// backend that will be used when applying the plan.
	PlanId         string
	PlanRefresh    bool   // PlanRefresh will do a refresh before a plan
	PlanOutPath    string // PlanOutPath is the path to save the plan
	PlanOutBackend *terraform.BackendState

	// Module settings specify the root module to use for operations.
	Module *module.Tree

	// Plan is a plan that was passed as an argument. This is valid for
	// plan and apply arguments but may not work for all backends.
	Plan *terraform.Plan

	// The options below are more self-explanatory and affect the runtime
	// behavior of the operation.
	Destroy   bool
	Targets   []string
	Variables map[string]interface{}

	// Input/output/control options.
	UIIn  terraform.UIInput
	UIOut terraform.UIOutput
}

// RunningOperation is the result of starting an operation.
type RunningOperation struct {
	// Context should be used to track Done and Err for errors.
	//
	// For implementers of a backend, this context should not wrap the
	// passed in context. Otherwise, canceling the parent context will
	// immediately mark this context as "done" but those aren't the semantics
	// we want: we want this context to be done only when the operation itself
	// is fully done.
	context.Context

	// Err is the error of the operation. This is populated after
	// the operation has completed.
	Err error

	// PlanEmpty is populated after a Plan operation completes without error
	// to note whether a plan is empty or has changes.
	PlanEmpty bool

	// State is the final state after the operation completed. Persisting
	// this state is managed by the backend. This should only be read
	// after the operation completes to avoid read/write races.
	State *terraform.State
}
