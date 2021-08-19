package stack_test

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// use for slice
const (
	bit  = 3
	mod  = 1<<bit - 1
	null = ^uintptr(0) // -1

)

type sueueNil *struct{}

// Interface use in stack,sueue testing
type Interface interface {
	Push(interface{}) bool
	Pop() (interface{}, bool)
	Top() (interface{}, bool)
	Size() int
	Empty() bool
}

// 单锁无限制链表栈
type SLStack struct {
	mu sync.Mutex

	len uint32
	top *listNode
}

func (s *SLStack) Empty() bool {
	return atomic.LoadUint32(&s.len) == 0
}

func (s *SLStack) Size() int {
	return int(atomic.LoadUint32(&s.len))
}

func (s *SLStack) Push(val interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val == nil {
		val = sueueNil(nil)
	}
	slot := newListNode(val)
	slot.next = s.top
	s.top = slot
	atomic.AddUint32(&s.len, 1)
	return true
}

func (s *SLStack) Top() (val interface{}, ok bool) {
	n := (*listNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.top))))
	if n == nil {
		return
	}
	val = n.load()
	return val, true
}

func (s *SLStack) Pop() (val interface{}, ok bool) {
	if s.Empty() {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Empty() {
		return
	}
	slot := s.top
	s.top = slot.next
	val = slot.load()
	if val == sueueNil(nil) {
		val = nil
	}
	slot.free()
	atomic.AddUint32(&s.len, ^uint32(0))
	return val, true
}

// 链表节点
type listNode struct {
	p    interface{}
	next *listNode
}

func newListNode(i interface{}) *listNode {
	ln := &listNode{}
	ln.store(i)
	return ln
}
func (n *listNode) load() interface{} {
	return n.p
}

func (n *listNode) store(i interface{}) {
	n.p = i
}
func (n *listNode) free() {
	n.p = nil
	n.next = nil
}
