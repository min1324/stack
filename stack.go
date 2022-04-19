package stack

import (
	"sync/atomic"
	"unsafe"
)

const (
	// initSize init when not provite cap,use initSize
	initSize = 1 << 8
)

type any = interface{}

// stackNil is used in stack to represent interface{}(nil).
// Since we use nil to represent empty slots, we need a sentinel value
// to represent nil.
type stackNil *struct{}

// Stack interface
type Stack interface {
	Interface

	// Top only returns the element at the top of the Stack.
	// It returns false if the Stack is empty.
	Top() (value any, ok bool)
}

// Interface stack base interface
type Interface interface {
	// Push adds val at the head of the Stack.
	// It returns false if the Stack is full.
	Push(value any) bool

	// Pop removes and returns the element at the head of the Stack.
	// It returns false if the Stack is empty.
	Pop() (value any, ok bool)
}

// New return an empty LockFree stack.
func New() Stack {
	return &LockFree{}
}

// LockFree a lock-free concurrent FILO stack.
type LockFree struct {
	// point to the latest value pushed.
	top unsafe.Pointer
}

// Push puts the given value at the top of the stack.
func (s *LockFree) Push(val interface{}) bool {
	if val == nil {
		val = stackNil(nil)
	}
	slot := &node{data: val}
	for {
		top := atomic.LoadPointer(&s.top)
		slot.next = top
		if atomic.CompareAndSwapPointer(&s.top, top, unsafe.Pointer(slot)) {
			return true
		}
	}
}

// Pop removes and returns the value at the top of the stack.
// It returns nil if the stack is empty.
func (s *LockFree) Pop() (val interface{}, ok bool) {
	var slot *node
	for {
		top := atomic.LoadPointer(&s.top)
		if top == nil {
			return
		}
		slot = (*node)(top)
		next := atomic.LoadPointer(&slot.next)
		runtime_procPin()
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			val = slot.data
			if val == stackNil(nil) {
				val = nil
			}
			atomic.StorePointer(&slot.next, nil)
			runtime_procUnpin()
			return val, true
		}
		runtime_procUnpin()
	}
}

// Top only returns the value at the top of the stack.
// It returns nil if the stack is empty.
func (s *LockFree) Top() (val interface{}, ok bool) {
	top := atomic.LoadPointer(&s.top)
	if top == nil {
		return
	}
	slot := (*node)(top)
	val = slot.data
	if val == stackNil(nil) {
		val = nil
	}
	return val, true
}

type node struct {
	data any
	next unsafe.Pointer
}

//go:linkname runtime_procPin runtime.procPin
func runtime_procPin()

//go:linkname runtime_procUnpin runtime.procUnpin
func runtime_procUnpin()
