package trace

import "time"

// PoolTrace is passed into radix.NewPool via radix.PoolWithTrace, and contains
// callbacks which will be triggered for specific events during the Pool's
// runtime.
//
// All callbacks are called synchronously.
type PoolTrace struct {
	// ConnCreated is called when the Pool creates a new connection. The
	// provided Err indicates whether the connection successfully completed.
	ConnCreated func(PoolConnCreated)

	// ConnClosed is called before closing the connection.
	ConnClosed func(PoolConnClosed)

	// DoCompleted is called after command execution. Must consider race condition
	// for manipulating variables in DoCompleted callback since DoComplete
	// function can be called in many go-routines.
	DoCompleted func(PoolDoCompleted)

	// InitCompleted is called after pool fills its connections
	InitCompleted func(PoolInitCompleted)
}

// PoolCommon contains information which is passed into all Pool-related
// callbacks.
type PoolCommon struct {
	// Network and Addr indicate the network/address the Pool was created with
	// (useful for differentiating different redis instances in a Cluster).
	Network, Addr string

	// PoolSize and BufferSize indicate the Pool size and buffer size that the
	// Pool was initialized with.
	PoolSize, BufferSize int
}

// PoolConnCreatedReason enumerates all the different reasons a connection might
// be created and trigger a ConnCreated trace.
type PoolConnCreatedReason string

// All possible values of PoolConnCreatedReason.
const (
	// PoolConnCreatedReasonInitialization indicates a connection was being
	// created during initialization of the Pool (i.e. within NewPool).
	PoolConnCreatedReasonInitialization PoolConnCreatedReason = "initialization"

	// PoolConnCreatedReasonRefill indicates a connection was being created
	// during a refill event (see radix.PoolRefillInterval).
	PoolConnCreatedReasonRefill PoolConnCreatedReason = "refill"

	// PoolConnCreatedReasonPoolEmpty indicates a connection was being created
	// because the Pool was empty and an Action requires one. See the
	// radix.PoolOnEmpty options.
	PoolConnCreatedReasonPoolEmpty PoolConnCreatedReason = "pool empty"
)

// PoolConnCreated is passed into the PoolTrace.ConnCreated callback whenever
// the Pool creates a new connection.
type PoolConnCreated struct {
	PoolCommon

	// The reason the connection was created.
	Reason PoolConnCreatedReason

	// How long it took to create the connection.
	ConnectTime time.Duration

	// If connection creation failed, this is the error it failed with.
	Err error
}

// PoolConnClosedReason enumerates all the different reasons a connection might
// be closed and trigger a ConnClosed trace.
type PoolConnClosedReason string

// All possible values of PoolConnClosedReason.
const (
	// PoolConnClosedReasonPoolClosed indicates a connection was closed because
	// the Close method was called on Pool.
	PoolConnClosedReasonPoolClosed PoolConnClosedReason = "pool closed"

	// PoolConnClosedReasonBufferDrain indicates a connection was closed due to
	// a buffer drain event. See radix.PoolOnFullBuffer.
	PoolConnClosedReasonBufferDrain PoolConnClosedReason = "buffer drained"

	// PoolConnClosedReasonPoolFull indicates a connection was closed due to
	// the Pool already being full. See The radix.PoolOnFull options.
	PoolConnClosedReasonPoolFull PoolConnClosedReason = "pool full"
)

// PoolConnClosed is passed into the PoolTrace.ConnClosed callback whenever the
// Pool closes a connection.
type PoolConnClosed struct {
	PoolCommon

	// AvailCount indicates the total number of connections the Pool is holding
	// on to which are available for usage at the moment the trace occurs.
	AvailCount int

	// The reason the connection was closed.
	Reason PoolConnClosedReason
}

// PoolDoCompleted is passed into the PoolTrace.DoCompleted callback whenever Pool finished to run
// Do function.
type PoolDoCompleted struct {
	PoolCommon

	// AvailCount indicates the total number of connections the Pool is holding
	// on to which are available for usage at the moment the trace occurs.
	AvailCount int

	// How long it took to send command.
	ElapsedTime time.Duration

	// This is the error returned from redis.
	Err error
}

// PoolInitCompleted is passed into the PoolTrace.InitCompleted callback whenever Pool initialized.
// This must be called once.
type PoolInitCompleted struct {
	PoolCommon

	// AvailCount indicates the total number of connections the Pool is holding
	// on to which are available for usage at the moment the trace occurs.
	AvailCount int

	// How long it took to fill all connections.
	ElapsedTime time.Duration
}
