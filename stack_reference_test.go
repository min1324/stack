package stack_test

import (
	"sync"

	"github.com/min1324/stack"
)

// use for slice
const (
	bit  = 3
	mod  = 1<<bit - 1
	null = ^uintptr(0) // -1

)

type stackNil *struct{}

// Interface use in stack,sueue testing
type Interface interface {
	stack.Stack
}

// 单锁无限制链表栈
type SLStack struct {
	mu sync.RWMutex

	top *listNode
}

func (s *SLStack) Push(val interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if val == nil {
		val = stackNil(nil)
	}
	s.top = &listNode{next: s.top, data: val}
	return true
}

func (s *SLStack) Top() (val interface{}, ok bool) {
	if s.top == nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.top == nil {
		return
	}
	return s.top.data, true
}

func (s *SLStack) Pop() (val interface{}, ok bool) {
	if s.top == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.top == nil {
		return
	}
	slot := s.top
	s.top = slot.next
	return slot.data, true
}

// 链表节点
type listNode struct {
	data interface{}
	next *listNode
}
