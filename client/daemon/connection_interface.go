package main

import "github.com/shadowmesh/shadowmesh/shared/protocol"

// ConnectionInterface provides a unified interface for both relay and P2P connections
type ConnectionInterface interface {
	// Start starts the connection
	Start() error

	// Stop stops the connection
	Stop() error

	// SendMessage sends a message
	SendMessage(msg *protocol.Message) error

	// ReceiveChannel returns the channel for receiving messages
	ReceiveChannel() <-chan *protocol.Message

	// ErrorChannel returns the channel for errors
	ErrorChannel() <-chan error

	// SetCallbacks sets connection callbacks
	SetCallbacks(onConnect func(), onDisconnect func(error), onMessage func(*protocol.Message))

	// IsConnected returns whether the connection is established
	IsConnected() bool
}

// Verify interfaces are implemented
var _ ConnectionInterface = (*ConnectionManager)(nil)
var _ ConnectionInterface = (*P2PConnectionManager)(nil)
