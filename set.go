package enjin

// A basic set data structure implementation
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

func (s set[T]) enumerate() (lst *[]T) {
	*lst = make([]T, len(s.vals))

	for i := range s.vals {
		*lst = append(*lst, i)
	}

	return
}

func build[T comparable](x ...T) (s *set[T]) {
    s.vals = make(map[T]bool, len(x))

	for _, v := range x {
		s.add(v)
	}

	return
}

func createFrom[T comparable](x []T) (s *set[T]) {
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
