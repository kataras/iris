package context

// A Handler responds to an HTTP request.
// It writes reply headers and data to the Context.ResponseWriter() and then return.
// Returning signals that the request is finished;
// it is not valid to use the Context after or concurrently with the completion of the Handler call.
//
// Depending on the HTTP client software, HTTP protocol version,
// and any intermediaries between the client and the iris server,
// it may not be possible to read from the Context.Request().Body after writing to the context.ResponseWriter().
// Cautious handlers should read the Context.Request().Body first, and then reply.
//
// Except for reading the body, handlers should not modify the provided Context.
//
// If Handler panics, the server (the caller of Handler) assumes that the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log, and hangs up the connection.
type Handler func(Context)

// Handlers is just a type of slice of []Handler.
//
// See `Handler` for more.
type Handlers []Handler

// HandlerList is a handler list splitted with middlewares at top and other handler at bottom
// It use an index for traking the bottom start
type HandlerList struct {
	handlers         Handlers
	bottomStartIndex int
}

// GetHandlers gets the handlers
func (hl *HandlerList) GetHandlers() Handlers {
	return hl.handlers
}

// AddToTop appends to the top the list
func (hl *HandlerList) AddToTop(handler Handler) *HandlerList {
	hl.handlers = append(hl.handlers, handler)

	// The bottom is not empty, we must insert handler the right place
	if len(hl.handlers) != hl.bottomStartIndex {
		// Shift the bottom, to the bottom of the list ;-)
		copy(hl.handlers[(hl.bottomStartIndex+1):], hl.handlers[hl.bottomStartIndex:])

		// Place new handler in new cell
		hl.handlers[hl.bottomStartIndex] = handler

		// Update the top of the bottom
		hl.bottomStartIndex++
	}

	return hl
}

// AddToTop concatenates the top with handler list
func (hl *HandlerList) AppendToTop(handlers Handlers) *HandlerList {
	// Size of this list used to concats
	size := len(hl.handlers)

	// Handler size, we need it for allocating, for updating bottom index,
	// and so to have the new bottom index
	handlerSize := len(handlers)

	if handlerSize > 0 {
		oldBottomIndex := hl.bottomStartIndex
		hl.bottomStartIndex += handlerSize

		// If the list is empty
		if size == 0 {
			// Just copy argument to this list
			hl.handlers = make(Handlers, len(handlers))
			copy(hl.handlers, handlers)
		} else {
			// Begin by allocating the new list (in all case, that's obvious that we must do it),
			// and copy the top of the list to the top of the new list
			newList := make(Handlers, size+handlerSize)
			copy(newList, hl.handlers)

			// If the bottom is not empty, we must shift the bottom to the bottom of the list to have the new space,
			// to have empty space needed for the insertion
			if size != oldBottomIndex {
				copy(newList[hl.bottomStartIndex:], newList[oldBottomIndex:])
			}

			// Place the new handlers to the new list at old bottom index,
			// it works even if the bottom is empty (cause in that case size == hl.bottomStartIndex)
			copy(newList[oldBottomIndex:], handlers)

			hl.handlers = newList
		}
	}

	return hl
}

// AddToBottom just appends to the list
func (hl *HandlerList) AddToBottom(handler Handler) *HandlerList {
	hl.handlers = append(hl.handlers, handler)

	return hl
}

// AddToBottom concatenates the bottom of the list with handler list, and so
// concats the whole list with handler list by using existing `joinHandlers` function
func (hl *HandlerList) AppendToBottom(handlers Handlers) *HandlerList {
	// Get old size
	size := len(hl.handlers)

	// Create a new slice of Handlers in order to store all handlers, the already handlers(Handlers) and the new
	newList := make(Handlers, size+len(handlers))

	// Copy the already Handlers to the just created
	copy(newList, hl.handlers)

	// Start from there we finish, and store the new Handlers too
	copy(newList[size:], handlers)

	// Save new list
	hl.handlers = newList

	return hl
}

// Copy just copies the instance by copying handler list
func (hl *HandlerList) Copy() *HandlerList {
	var newList Handlers

	if len(hl.handlers) > 0 {
		newList = make(Handlers, len(hl.handlers))
		copy(newList, hl.handlers)
	}

	return &HandlerList{
		handlers:         newList,
		bottomStartIndex: hl.bottomStartIndex,
	}
}

// NewHandlerList creates a new handler list
func NewHandlerList() *HandlerList {
	return new(HandlerList)
}
