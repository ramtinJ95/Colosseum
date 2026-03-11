package tmux

import (
	"context"
	"fmt"
)

type MockCall struct {
	Args        []string
	Interactive bool
}

type MockResponse struct {
	Output string
	Err    error
}

type MockCommander struct {
	Calls     []MockCall
	Responses []MockResponse
	callIndex int
}

func NewMockCommander(responses ...MockResponse) *MockCommander {
	return &MockCommander{
		Responses: responses,
	}
}

func (m *MockCommander) Run(_ context.Context, args ...string) (string, error) {
	m.Calls = append(m.Calls, MockCall{Args: args})

	if m.callIndex >= len(m.Responses) {
		return "", fmt.Errorf("mock: no response configured for call %d", m.callIndex)
	}

	resp := m.Responses[m.callIndex]
	m.callIndex++
	return resp.Output, resp.Err
}

func (m *MockCommander) RunInteractive(_ context.Context, args ...string) error {
	m.Calls = append(m.Calls, MockCall{Args: args, Interactive: true})

	if m.callIndex >= len(m.Responses) {
		return fmt.Errorf("mock: no response configured for call %d", m.callIndex)
	}

	resp := m.Responses[m.callIndex]
	m.callIndex++
	return resp.Err
}
