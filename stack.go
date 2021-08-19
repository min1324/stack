package stack

import (
	"sync/atomic"
	"unsafe"
)

// New return an empty stack.
func New() *Stack {
	return &Stack{}
}

// Stack a lock-free concurrent FILO stack.
type Stack struct {
	len uint32         // stack value num.
	top unsafe.Pointer // point to the latest value pushed.
}

func (s *Stack) Empty() bool {
	return atomic.LoadUint32(&s.len) == 0
}

// Size stack element's number
func (s *Stack) Size() int {
	return int(atomic.LoadUint32(&s.len))
}

// Push puts the given value at the top of the stack.
func (s *Stack) Push(val interface{}) bool {
	slot := newPrtNode(val)
	for {
		slot.next = atomic.LoadPointer(&s.top)
		if atomic.CompareAndSwapPointer(&s.top, slot.next, unsafe.Pointer(slot)) {
			atomic.AddUint32(&s.len, 1)
			break
		}
	}
	return true
}

// Pop removes and returns the value at the top of the stack.
// It returns nil if the stack is empty.
func (s *Stack) Pop() (val interface{}, ok bool) {
	var slot *ptrNode
	for {
		top := atomic.LoadPointer(&s.top)
		if top == nil {
			return
		}
		slot = (*ptrNode)(top)
		next := atomic.LoadPointer(&slot.next)
		if atomic.CompareAndSwapPointer(&s.top, top, next) {
			atomic.AddUint32(&s.len, ^uint32(0))
			break
		}
	}
	val = slot.load()
	slot.free()
	return val, true
}

// Top only returns the value at the top of the stack.
// It returns nil if the stack is empty.
func (s *Stack) Top() (val interface{}, ok bool) {
	top := atomic.LoadPointer(&s.top)
	if top == nil {
		return
	}
	slot := (*ptrNode)(top)
	val = slot.load()
	return val, true
}

type ptrNode struct {
	entry
	next unsafe.Pointer
}

func newPrtNode(i interface{}) *ptrNode {
	p := &ptrNode{}
	p.store(i)
	return p
}

// entry stack element
type entry struct {
	p unsafe.Pointer
}

func (n *entry) load() interface{} {
	p := atomic.LoadPointer(&n.p)
	if p == nil {
		return nil
	}
	return *(*interface{})(p)
}

func (n *entry) store(i interface{}) {
	atomic.StorePointer(&n.p, unsafe.Pointer(&i))
}

func (n *entry) free() {
	atomic.StorePointer(&n.p, nil)
}
