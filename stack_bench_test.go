package stack_test

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/min1324/stack"
)

/*
1<< 20~28
1048576		20
2097152		21
4194304		22
8388608		23
16777216	24
33554432	25
67108864	26
134217728	27
268435456	28
*/
const prevPushSize = 1 << 20 // queue previous Push

type benchS struct {
	setup func(*testing.B, Interface)
	perG  func(b *testing.B, pb *testing.PB, i int, m Interface)
}

func benchSMap(b *testing.B, benchS benchS) {
	for _, m := range [...]Interface{
		// // stack
		&stack.LLStack{},
		&SLStack{},
	} {
		b.Run(fmt.Sprintf("%T", m), func(b *testing.B) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(Interface)
			if benchS.setup != nil {
				benchS.setup(b, m)
			}

			b.ResetTimer()

			var i int64
			b.RunParallel(func(pb *testing.PB) {
				id := int(atomic.AddInt64(&i, 1) - 1)
				benchS.perG(b, pb, (id * b.N), m)
			})
			// free

		})
	}
}

func BenchmarkPush(b *testing.B) {
	benchSMap(b, benchS{
		setup: func(_ *testing.B, m Interface) {
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Push(i)
			}
		},
	})
}

func BenchmarkPop(b *testing.B) {
	// 由于预存的数量<出队数量，无法准确测试dequeue
	const prevsize = 1 << 20
	benchSMap(b, benchS{
		setup: func(b *testing.B, m Interface) {
			for i := 0; i < prevsize; i++ {
				m.Push(i)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Pop()
			}
		},
	})
}

func BenchmarkStackBalance(b *testing.B) {

	benchSMap(b, benchS{
		setup: func(_ *testing.B, m Interface) {
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			for ; pb.Next(); i++ {
				m.Push(1)
				m.Pop()
				// b.Log("Push:", m.Size(), m.Push(i+10), m.Size(), i)
				// size := m.Size()
				// v, ok := m.Pop()
				// b.Logf("pop:%d,%v,%v,%d,%d", size, v, ok, m.Size(), i)
			}
		},
	})
}

func BenchmarkStackInterlace(b *testing.B) {
	const mark = 1<<2 - 1
	benchSMap(b, benchS{
		setup: func(_ *testing.B, m Interface) {
			for i := 0; i < prevPushSize; i++ {
				m.Push(i)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m Interface) {
			j := 0
			for ; pb.Next(); i++ {
				j += (i & mark)
				if j&1 == 0 {
					m.Push(i)
				} else {
					m.Pop()
				}
			}
		},
	})
}
