package stack

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	// DefaultSize init when not provite cap,use DefaultSize
	DefaultSize = 1 << 8
)

// stackNil is used in queue to represent interface{}(nil).
// Since we use nil to represent empty slots, we need a sentinel value
// to represent nil.
type stackNil *struct{}

// Stack interface
type Stack interface {
	Init()
	OnceInit(cap int)
	Cap() int
	Empty() bool
	Full() bool
	Size() int
	Push(val interface{}) bool
	Pop() (val interface{}, ok bool)
	Top() (val interface{}, ok bool)
}

// New return an empty LFStack.
func New() Stack {
	return &LFStack{}
}

// NewCap return an empty LAStack with init cap.
// if cap<1,will use DefaultSize
func NewCap(cap int) Stack {
	var s LAStack
	s.OnceInit(cap)
	return &s
}

// LFStack a lock-free concurrent FILO stack.
type LFStack struct {
	len uint32         // stack value num.
	top unsafe.Pointer // point to the latest value pushed.
}

// OnceInit initialize queue use cap
// it only execute once time.
// if cap<1, will use DefaultSize.
func (s *LFStack) OnceInit(cap int) {}

// Init initialize queue use DefaultSize: 256
// it only execute once time.
func (s *LFStack) Init() {}

// Cap return queue's cap
func (s *LFStack) Cap() int {
	return 1<<31 - 1
}

// Empty return queue if empty
func (s *LFStack) Empty() bool {
	return atomic.LoadUint32(&s.len) == 0
}

// Full return queue if full
func (s *LFStack) Full() bool {
	return atomic.LoadUint32(&s.len) >= (1<<31 - 1)
}

// Size return current number in stack
func (s *LFStack) Size() int {
	return int(atomic.LoadUint32(&s.len))
}

// Push puts the given value at the top of the stack.
func (s *LFStack) Push(val interface{}) bool {
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
func (s *LFStack) Pop() (val interface{}, ok bool) {
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
func (s *LFStack) Top() (val interface{}, ok bool) {
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
			cap = DefaultSize
		}
		atomic.StoreUint32(&s.len, 0)
		atomic.StoreUint32(&s.cap, uint32(cap))
		s.data = make([]entry, cap)
	})
}

// OnceInit initialize queue use cap
// it only execute once time.
// if cap<1, will use DefaultSize.
func (s *LAStack) OnceInit(cap int) {
	s.onceInit(cap)
}

// Init initialize queue use DefaultSize: 256
// it only execute once time.
func (s *LAStack) Init() {
	s.onceInit(DefaultSize)
}

// Cap return queue's cap
func (s *LAStack) Cap() int {
	return int(atomic.LoadUint32(&s.cap))
}

// Empty return queue if empty
func (s *LAStack) Empty() bool {
	return atomic.LoadUint32(&s.len) == 0
}

// Full return queue if full
func (s *LAStack) Full() bool {
	return atomic.LoadUint32(&s.len) == atomic.LoadUint32(&s.cap)
}

// Size return current number in stack
func (s *LAStack) Size() int {
	return int(atomic.LoadUint32(&s.len))
}

// 根据enID,deID获取进队，出队对应的slot
func (s *LAStack) getSlot(id uint32) *entry {
	return &s.data[id]
}

// Push puts the given value at the top of the stack.
// it return true if success,or false if queue full.
func (s *LAStack) Push(val interface{}) bool {
	s.Init()
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
			if val == nil {
				val = stackNil(nil)
			}
			slot.store(val)
			break
		}
	}
	return true
}

// Pop removes and returns the value at the top of the stack.
// it return true if success,or false if stack empty.
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

// Pop only returns the value at the top of the stack.
// it return true if success,or false if stack empty.
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
