package main

import "golang.org/x/exp/constraints"

func nonempty[T comparable](xs []T) []T {
	var ret []T
	var zero T
	for _, x := range xs {
		if x != zero {
			ret = append(ret, x)
		}
	}
	return ret
}

func map1[T, U any](f func(T) U, xs []T) []U {
	ret := make([]U, len(xs))
	for i, x := range xs {
		ret[i] = f(x)
	}
	return ret
}

func powerset[T any](xs []T) [][]T {
	if len(xs) == 0 {
		return [][]T{{}}
	}
	r := powerset(xs[1:])
	ret := make([][]T, 2*len(r))
	for i, ys := range r {
		ret[2*i] = make([]T, len(ys)+1)
		ret[2*i][0] = xs[0]
		copy(ret[2*i][1:], ys)
		ret[2*i+1] = ys
	}
	return ret
}

type numeric interface {
	constraints.Integer | constraints.Float
}

func sum[T numeric](xs []T) T {
	var acc T
	for _, x := range xs {
		acc += x
	}
	return acc
}

func min[T numeric](xs ...T) T {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func max[T numeric](xs ...T) T {
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

func ternary[T any](cond bool, th, el T) T {
	if cond {
		return th
	} else {
		return el
	}
}
