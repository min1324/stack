package stack

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type stackNil *struct{}

// New return an empty stack.
func New() *Stack {
	return &Stack{}
}

// Stack a lock-free concurrent FILO stack.
type Stack struct {
	len uint32         // stack value num.
	top unsafe.Pointer // point to the latest value pushed.
}

// Empty return stack if empty
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

const (
	initSize = 1 << 8
)

// LAStack a lock-free concurrent FILO array stack.
type LAStack struct {
	once sync.Once
	len  uint32
	cap  uint32
	data []entry
}

// 一次性初始化
func (s *LAStack) onceInit(cap int) {
	s.once.Do(func() {
		if cap < 1 {
			cap = initSize
		}
		s.len = 0
		s.cap = uint32(cap)
		s.data = make([]entry, cap)
	})
}

// Init初始化长度为: DefauleSize.
func (s *LAStack) Init() {
	s.onceInit(initSize)
}

// InitWith初始化长度为cap的queue,
// 如果未提供，则使用默认值: DefauleSize.
func (s *LAStack) OnceInit(cap int) {
	s.onceInit(cap)
}

func (s *LAStack) Cap() int {
	return int(atomic.LoadUint32(&s.cap))
}

// Size stack element's number
func (s *LAStack) Empty() bool {
	return atomic.LoadUint32(&s.len) == 0
}

// Size stack element's number
func (s *LAStack) Full() bool {
	return atomic.LoadUint32(&s.len) == atomic.LoadUint32(&s.cap)
}

// Size stack element's number
func (s *LAStack) Size() int {
	return int(atomic.LoadUint32(&s.len))
}

func (s *LAStack) getSlot(id uint32) *entry {
	return &s.data[id]
}

// Push puts the given value at the top of the stack.
func (s *LAStack) Push(val interface{}) bool {
	s.Init()
	if val == nil {
		val = stackNil(nil)
	}
	cap := atomic.LoadUint32(&s.cap)
	for {
		top := atomic.LoadUint32(&s.len)
		if top == cap {
			return false
		}
		slot := s.getSlot(top)
		if slot.load() != nil {
			return false
		}
		if casUint32(&s.len, top, top+1) {
			slot.store(val)
			break
		}
	}
	return true
}

// Pop removes and returns the value at the top of the stack.
// It returns nil if the stack is empty.
func (s *LAStack) Pop() (val interface{}, ok bool) {
	s.Init()
	var slot *entry
	for {
		top := atomic.LoadUint32(&s.len)
		if top == 0 {
			return nil, false
		}
		slot = s.getSlot(top - 1)
		val = slot.load()
		if val == nil {
			return nil, false
		}
		if casUint32(&s.len, top, top-1) {
			slot.free()
			break
		}
	}
	if val == stackNil(nil) {
		val = nil
	}
	return val, true
}

func (s *LAStack) Top() (val interface{}, ok bool) {
	top := atomic.LoadUint32(&s.len)
	if top == 0 {
		return
	}
	slot := s.getSlot(top - 1)
	val = slot.load()
	if val == stackNil(nil) {
		val = nil
	}
	return val, true
}

func casUint32(p *uint32, old, new uint32) bool {
	return atomic.CompareAndSwapUint32(p, old, new)
}
