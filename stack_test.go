package stack_test

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/min1324/stack"
)

type stackStruct struct {
	setup func(*testing.T, Interface)
	perG  func(*testing.T, Interface)
}

func stackMap(t *testing.T, test stackStruct) {
	for _, m := range [...]Interface{
		&stack.LockFree{},
		&SLStack{},
		// &stack.LAStack{},
	} {
		t.Run(fmt.Sprintf("%T", m), func(t *testing.T) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(Interface)
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

			if v, ok := s.Top(); ok || v != nil {
				t.Fatalf("init Top != nil :%v,%v", v, ok)
			}

			if v, ok := s.Pop(); ok {
				t.Fatalf("init Pop != nil :%v", v)
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

func TestConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	goNum := runtime.NumCPU()
	var max = 1000000
	var q stack.LockFree

	var args = make([]uint32, max)
	var result sync.Map
	var done uint32

	for i := range args {
		args[i] = uint32(i)
	}
	// Pop
	wg.Add(goNum)
	for i := 0; i < goNum; i++ {
		go func() {
			defer wg.Done()
			for {
				if _, ok := q.Top(); atomic.LoadUint32(&done) == 1 && !ok {
					break
				}
				v, ok := q.Pop()
				if ok {
					result.Store(v, v)
				}
			}
		}()
	}
	// Push
	wg.Add(goNum)
	var gbCount uint32 = 0
	for i := 0; i < goNum; i++ {
		go func() {
			defer wg.Done()
			for {
				c := atomic.AddUint32(&gbCount, 1)
				if c >= uint32(max) {
					break
				}
				for !q.Push(c) {
				}
			}
		}()
	}
	// wait until finish Push
	for {
		c := atomic.LoadUint32(&gbCount)
		if c > uint32(max) {
			break
		}
		runtime.Gosched()
	}
	// wait until Pop
	atomic.StoreUint32(&done, 1)
	wg.Wait()

	// check
	for i := 1; i < max; i++ {
		e := args[i]
		v, ok := result.Load(e)
		if !ok {
			t.Errorf("err miss:%v,ok:%v,e:%v ", v, ok, e)
		}
	}
}

func TestLFQueue(t *testing.T) {
	var d stack.LockFree
	testPoolPop(t, &d)
}

func testPoolPop(t *testing.T, d stack.Stack) {
	const P = 10
	var N int = 2e6
	if testing.Short() {
		N = 1e3
	}
	have := make([]int32, N)
	var stop int32
	var wg sync.WaitGroup
	record := func(val int) {
		atomic.AddInt32(&have[val], 1)
		if val == N-1 {
			atomic.StoreInt32(&stop, 1)
		}
	}

	// Start P-1 consumers.
	for i := 1; i < P; i++ {
		wg.Add(1)
		go func() {
			fail := 0
			for atomic.LoadInt32(&stop) == 0 {
				val, ok := d.Pop()
				if ok {
					fail = 0
					record(val.(int))
				} else {
					// Speed up the test by
					// allowing the pusher to run.
					if fail++; fail%100 == 0 {
						runtime.Gosched()
					}
				}
			}
			wg.Done()
		}()
	}

	// Start 1 producer.
	nPopHead := 0
	wg.Add(1)
	go func() {
		for j := 0; j < N; j++ {
			for !d.Push(j) {
				// Allow a popper to run.
				runtime.Gosched()
			}
			if j%10 == 0 {
				val, ok := d.Pop()
				if ok {
					nPopHead++
					record(val.(int))
				}
			}
		}
		wg.Done()
	}()
	wg.Wait()

	// Check results.
	for i, count := range have {
		if count != 1 {
			t.Errorf("expected have[%d] = 1, got %d", i, count)
		}
	}
	// Check that at least some PopHeads succeeded. We skip this
	// check in short mode because it's common enough that the
	// queue will stay nearly empty all the time and a PopTail
	// will happen during the window between every PushHead and
	// PopHead.
	if !testing.Short() && nPopHead == 0 {
		t.Errorf("popHead never succeeded")
	}
}
