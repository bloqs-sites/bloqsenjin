package enjin

import "fmt"

// A basic et data structure implementation
type set[T comparable] struct {
	vals map[T]bool
}

func (s set[T]) isElementOf(key T) bool {
	return bool(s.vals[key])
}

func (s set[T]) isEmpty() bool {
	return len(s.vals) == 0
}

func (s set[T]) size() int {
	return len(s.vals)
}

func (s set[T]) enumerate() (lst []*T) {
	fmt.Println("enumerate start")
	lst = make([]*T, len(s.vals))
	fmt.Println(s.vals, lst)

	i := 0
	for v := range s.vals {
		fmt.Println(v, i)
		lst[i] = &v
		i++
	}

	fmt.Println("enumerate end")
	return
}

func buildSet[T comparable](x ...T) (s set[T]) {
	s.vals = make(map[T]bool, len(x))

	for _, v := range x {
		s.add(v)
	}

	return
}

func (s *set[T]) add(x T) {
	s.vals[x] = true
}

func (s *set[T]) remove(x T) {
	s.vals[x] = false
}
