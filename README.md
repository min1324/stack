# stack

[![Build Status](https://travis-ci.com/min1324/stack.svg?branch=main)](https://travis-ci.com/min1324/stack) [![codecov](https://codecov.io/gh/min1324/stack/branch/main/graph/badge.svg)](https://codecov.io/gh/min1324/stack) [![GoDoc](https://godoc.org/github.com/min1324/stack?status.png)](https://godoc.org/github.com/min1324/stack) [![Go Report Card](https://goreportcard.com/badge/github.com/min1324/stack)](https://goreportcard.com/report/github.com/min1324/stack)

-----

栈(`stack`)是非常常用的一个数据结构，它只允许在栈的栈顶（`head`）进行操作，是一种典型的FILO数据结构。[**lock-free**][1]的算法都是通过`CAS`操作实现的。

stack接口：

```go
type stack interface {
	Push(interface{}) bool
	Pop() (val interface{}, ok bool)
}
```

**Push**

将val加入栈顶，返回是否成功。

**Pop**

取出栈顶val，返回val和是否成功，如果不成功，val为nil。



-----




[1]: https://www.cs.rochester.edu/u/scott/papers/1996_PODC_queues.pdf

