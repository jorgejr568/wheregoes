package set

type Set[T comparable] interface {
	Add(T)
	Remove(T)
	Contains(T) bool
	Len() int
	Values() []T
}

type set[T comparable] struct {
	m map[T]struct{}
}

func (s set[T]) Add(t T) {
	s.m[t] = struct{}{}
}

func (s set[T]) Remove(t T) {
	delete(s.m, t)
}

func (s set[T]) Contains(t T) bool {
	_, ok := s.m[t]
	return ok
}

func (s set[T]) Len() int {
	return len(s.m)
}

func (s set[T]) Values() []T {
	values := make([]T, 0, len(s.m))
	for k := range s.m {
		values = append(values, k)
	}
	return values
}

func New[T comparable]() Set[T] {
	return &set[T]{
		m: make(map[T]struct{}),
	}
}
