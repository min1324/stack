package stack_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"unsafe"

	"github.com/min1324/stack"
)

type mapOp string

const (
	opPush = mapOp("Push")
	opPop  = mapOp("Pop")
)

var mapOps = [...]mapOp{opPush, opPop}

// mapCall is a quick.Generator for calls on mapInterface.
type mapCall struct {
	op mapOp
	k  interface{}
}

type mapResult struct {
	value interface{}
	ok    bool
}

func (c mapCall) apply(m Interface) (interface{}, bool) {
	switch c.op {
	case opPush:
		return c.k, m.Push(c.k)
	case opPop:
		return m.Pop()
	default:
		panic("invalid mapOp")
	}
}

func randValue(r *rand.Rand) interface{} {
	b := make([]byte, r.Intn(4))
	for i := range b {
		b[i] = 'a' + byte(rand.Intn(26))
	}
	return string(b)
}

func (mapCall) Generate(r *rand.Rand, size int) reflect.Value {
	c := mapCall{op: mapOps[rand.Intn(len(mapOps))], k: randValue(r)}
	return reflect.ValueOf(c)
}

func applyCalls(m Interface, calls []mapCall) (results []mapResult, final map[interface{}]interface{}) {
	for _, c := range calls {
		v, ok := c.apply(m)
		results = append(results, mapResult{v, ok})
	}

	final = make(map[interface{}]interface{})

	for m.Size() > 0 {
		v, ok := m.Pop()
		final[v] = ok
	}
	return results, final
}

func applyStack(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	q := stack.New()
	return applyCalls(q, calls)
}

func applyMutexStack(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	var q SLStack
	return applyCalls(&q, calls)
}

func applyLAStack(calls []mapCall) ([]mapResult, map[interface{}]interface{}) {
	var q stack.LAStack
	return applyCalls(&q, calls)
}

func TestMatchesMutex(t *testing.T) {
	if err := quick.CheckEqual(applyStack, applyMutexStack, nil); err != nil {
		t.Error(err)
	}
}

func TestMatchesLA(t *testing.T) {
	if err := quick.CheckEqual(applyStack, applyLAStack, nil); err != nil {
		t.Error(err)
	}
}

type stackStruct struct {
	setup func(*testing.T, Interface)
	perG  func(*testing.T, Interface)
}

func stackMap(t *testing.T, test stackStruct) {
	for _, m := range [...]Interface{
		&stack.LFStack{},
		&SLStack{},
		&stack.LAStack{},
	} {
		t.Run(fmt.Sprintf("%T", m), func(t *testing.T) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(Interface)
			if s, ok := m.(*stack.LAStack); ok {
				s.OnceInit(prevPushSize)
			}
			if test.setup != nil {
				test.setup(t, m)
			}
			test.perG(t, m)
		})
	}
}

func TestStackInit(t *testing.T) {
	stackMap(t, stackStruct{
		setup: func(t *testing.T, s Interface) {
		},
		perG: func(t *testing.T, s Interface) {
			// 初始化测试，
			if s.Size() != 0 {
				t.Fatalf("init size != 0 :%d", s.Size())
			}

			if !s.Empty() {
				t.Fatalf("init not Empty")
			}

			if v, ok := s.Top(); ok || v != nil {
				t.Fatalf("init Top != nil :%v,%v", v, ok)
			}

			if v, ok := s.Pop(); ok {
				t.Fatalf("init Pop != nil :%v", v)
			}

			// Push,Pop测试
			p := 1
			b := s.Push(p)
			if s.Size() != 1 || !b {
				t.Fatalf("after Push err,size!=1,%d,%v", s.Size(), b)
			}
			if v, ok := s.Top(); !ok || v != p {
				t.Fatalf("Push want:%d, real:%v", p, v)
			}
			if v, ok := s.Pop(); !ok || v != p {
				t.Fatalf("Push want:%d, real:%v", p, v)
			}

			// size 测试
			var n = 10
			var esum int
			for i := 0; i < n; i++ {
				if s.Push(i) {
					esum++
				}
			}
			if s.Size() != esum {
				t.Fatalf("Size want:%d, real:%v", esum, s.Size())
			}
			for {
				_, ok := s.Pop()
				if !ok {
					break
				}
			}

			// 储存顺序测试,数组队列可能满
			// stack顺序反过来
			array := [...]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
			sum := 0
			for i := range array {
				if s.Push(i) {
					array[sum] = sum // stack用这种方式
					sum += 1
				}

			}
			for i := sum - 1; i >= 0; i-- {
				v, ok := s.Pop()
				if !ok || v != array[i] {
					t.Fatalf("array want:%d, real:%v,size:%d,%v", array[i], v, sum, ok)
				}
			}

			// 空值测试
			var nullPtrs = unsafe.Pointer(nil)
			s.Push(nullPtrs)

			if v, ok := s.Pop(); !ok || nullPtrs != v {
				t.Fatalf("Push nil want:%v, real:%v", nullPtrs, v)
			}
			var null = new(interface{})
			s.Push(null)
			if v, ok := s.Pop(); !ok || null != v {
				t.Fatalf("Push nil want:%v, real:%v", null, v)
			}
		},
	})
}
