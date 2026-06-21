// Package internal is a stand-in for go.temporal.io/sdk/internal. The real SDK
// declares the core types here and re-publishes them from the workflow package
// as aliases (type Context = internal.Context), so the fixtures mirror that
// shape to exercise the analyzer against the SDK's real type identities.
package internal

type Context interface{ isWorkflowContext() }

// MockCallWrapper stands in for the testsuite type of the same name, declared
// here and re-published from testsuite as an alias. It carries just enough to
// let fixtures chain .Return(...).Once() after a mock setup.
type MockCallWrapper struct{}

func (c *MockCallWrapper) Return(returnArguments ...any) *MockCallWrapper { return c }

func (c *MockCallWrapper) Once() *MockCallWrapper { return c }

// TestWorkflowEnvironment stands in for the testsuite type of the same name. As
// with Context, the real SDK declares it here and re-publishes it from testsuite
// as an alias. OnActivity/OnWorkflow take the target as interface{} and the
// matchers as variadic interface{} -- the same type erasure the analyzer's
// strict-tests check inspects.
type TestWorkflowEnvironment struct{}

func (e *TestWorkflowEnvironment) OnActivity(activity any, args ...any) *MockCallWrapper {
	return &MockCallWrapper{}
}

func (e *TestWorkflowEnvironment) OnWorkflow(workflow any, args ...any) *MockCallWrapper {
	return &MockCallWrapper{}
}
