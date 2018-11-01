package opencensus

import (
	"net/http"

	"github.com/opentracing/opentracing-go"
)

const defaultMaxHandlerCount = 5

// Subsetted from https://github.com/aws/aws-sdk-go/blob/master/aws/request/handlers.go

// A NamedHandler is a struct that contains a name and function callback.
type NamedHandler struct {
	Name string
	Func func(*http.Request, opentracing.Span)
}

// A HandlerList manages zero or more handlers in a list.
type HandlerList struct {
	list []NamedHandler
}

// NewHandlerList returns a default HandlerList.
func NewHandlerList() *HandlerList {
	return NewHandlerListWithSize(defaultMaxHandlerCount)
}

// NewHandlerListWithSize returns a HandlerList with size.
func NewHandlerListWithSize(size int) *HandlerList {
	l := make([]NamedHandler, 0, size)
	l = append(l, NamedHandler{
		Name: "ClientRequestInfoAsTag",
		Func: func(req *http.Request, span opentracing.Span) {
			tagHTTPClientRequest(span, req)
		},
	})
	return &HandlerList{
		list: l,
	}
}

// IsFull returns a flag to indicate whether HandlerList is full.
func (l *HandlerList) IsFull() bool {
	return l.size() == l.cap()
}

// IsEmpty returns a flag to indicate whether HandlerList is empty.
func (l *HandlerList) IsEmpty() bool {
	return l.size() == 0
}

// PushBackNamed pushes named handler f to the back of the handler list.
func (l *HandlerList) PushBackNamed(h NamedHandler) {
	if !l.IsFull() {
		l.list = append(l.list, h)
	}
}

// Run executes all handlers in the list with a given request object.
func (l *HandlerList) Run(req *http.Request, span opentracing.Span) {
	if !l.IsEmpty() {
		for _, h := range l.list {
			h.Func(req, span)
		}
	}
}

// size returns the number of handlers.
func (l *HandlerList) size() int {
	return len(l.list)
}

func (l *HandlerList) cap() int {
	return cap(l.list)
}
